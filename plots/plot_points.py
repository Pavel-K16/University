#!/usr/bin/env python3
"""
Генерация HTML-страницы с графиками траекторий и суммарной энергии.

Данные берутся из:
  ../wolfram/paramsAndPoints/graph_points*.txt   (t, x)
  ../wolfram/paramsAndPoints/sumEnergyPoints.txt (t, E)
  ../wolfram/paramsAndPoints/energy_points*.txt (t, E по узлу) — только при PARALLEL=false
  ../wolfram/paramsAndPoints/d_energy_points*.txt (t, dE/dt по узлу) — только при PARALLEL=false

Результаты сохраняются в:
  plots/trajectories.png
  plots/sum_energy.png
  plots/node_energy.png
  plots/node_d_energy.png
  plots/analytical_dadt.png — аналитика dA/dt по среднему δ (только PARALLEL=false)
  plots/d_amplitudes.png — производная амплитуды из d_amplitude_points*.txt (только PARALLEL=false)
  plots/index.html
  plots/decrement_map_node*_interactive.html — интерактивные карты декремента (клик по точке)

Переменные окружения (опционально):
  A0 — коэффициент A0 в аналитической формуле dA/dt (по умолчанию 1; например export A0=-1).
  ANALYTICAL_DT_SOURCE — peak (по умолчанию): Δt = среднее расстояние между пиками
    из amplitude_points*.txt; integrator — брать dt из конфига times.dt.
"""

from pathlib import Path
import html as html_lib
import json
import os
import re

import matplotlib
import matplotlib.pyplot as plt
from matplotlib.colors import Normalize, to_hex
import numpy as np


def env_bool(name: str, default: bool = False) -> bool:
    raw = os.environ.get(name)
    if raw is None:
        return default
    return raw.strip().lower() in {"1", "true", "yes", "on"}


# Выставляется из scripts/run_and_plot.sh вторым аргументом (или вручную: export PARALLEL=true).
PARALLEL_ONLY_DECREMENT_MAPS = env_bool("PARALLEL", default=False)

# При генерации PNG: тёмный фон и светлые оси (export PLOT_DARK=true).
PLOT_DARK_EXPORT = env_bool("PLOT_DARK", default=False)


# Верхняя граница по модулю для столбцов из txt: отсекаем огромные конечные числа, из‑за которых
# matplotlib ломается на tight_layout (tick locator / arange).
_PLOT_COL_ABS_MAX = 1e100

# Совпадают с zeroMaxs и oneMax в internal/numMethods/utils/damping_decrement.go.
_DECREMENT_ZERO_MAXS_SENTINEL = 123456.123456
_DECREMENT_ONE_MAX_SENTINEL = 1232323.1232323


def _mask_plottable_2cols(c0: np.ndarray, c1: np.ndarray) -> np.ndarray:
    """Конечные значения и |·| не выше порога — чтобы оси и тики оставались численно устойчивыми."""
    if c0.size == 0:
        return np.array([], dtype=bool)
    lim = _PLOT_COL_ABS_MAX
    return (
        np.isfinite(c0)
        & np.isfinite(c1)
        & (np.abs(c0) <= lim)
        & (np.abs(c1) <= lim)
    )


def _mask_plottable_3cols(c0: np.ndarray, c1: np.ndarray, c2: np.ndarray) -> np.ndarray:
    if c0.size == 0:
        return np.array([], dtype=bool)
    lim = _PLOT_COL_ABS_MAX
    return (
        np.isfinite(c0)
        & np.isfinite(c1)
        & np.isfinite(c2)
        & (np.abs(c0) <= lim)
        & (np.abs(c1) <= lim)
        & (np.abs(c2) <= lim)
    )


def safe_tight_layout() -> None:
    """tight_layout иногда падает на экстремальных пределах осей — тогда ослабляем разметку."""
    try:
        plt.tight_layout()
    except Exception as ex:
        print(f"Предупреждение: tight_layout пропущен ({type(ex).__name__}: {ex})")
        try:
            plt.subplots_adjust(left=0.10, right=0.96, top=0.92, bottom=0.12)
        except Exception:
            pass


def load_two_column_txt(path: Path):
    """
    Ожидается файл с двумя столбцами: t, value.
    Возвращает (t, y) как numpy-массивы.
    Строки с nan/inf или чрезмерно большими |·| в любом столбце отбрасываются.
    """
    if path.stat().st_size == 0:
        return np.array([]), np.array([])
    data = np.loadtxt(path)
    if np.size(data) == 0:
        return np.array([]), np.array([])
    if data.ndim == 1:
        data = data[None, :]
    t = data[:, 0]
    y = data[:, 1]
    m = _mask_plottable_2cols(t, y)
    return t[m], y[m]


def load_three_column_txt(path: Path):
    """
    Ожидается файл с тремя столбцами: x, y, value.
    Возвращает (x, y, v) как numpy-массивы.
    Строки с nan/inf или чрезмерно большими |·| отбрасываются.
    """
    if path.stat().st_size == 0:
        return np.array([]), np.array([]), np.array([])
    data = np.loadtxt(path)
    if np.size(data) == 0:
        return np.array([]), np.array([]), np.array([])
    if data.ndim == 1:
        data = data[None, :]
    x = data[:, 0]
    y = data[:, 1]
    v = data[:, 2]
    m = _mask_plottable_3cols(x, y, v)
    return x[m], y[m], v[m]


def _viridis_cmap():
    try:
        return matplotlib.colormaps["viridis"]
    except (AttributeError, KeyError):
        return matplotlib.cm.get_cmap("viridis")


def _write_decrement_map_interactive_html(
    plots_dir: Path,
    node_id: str,
    bx: np.ndarray,
    fy: np.ndarray,
    delta: np.ndarray,
) -> str | None:
    """
    Автономный HTML+SVG+JS: клик по точке показывает koef1, koef2 и delta.
    Маркеры и цвета согласованы со статической картой (круг/квадрат/треугольники).
    """
    if bx.size == 0:
        return None

    mask_z = np.isclose(delta, _DECREMENT_ZERO_MAXS_SENTINEL, rtol=0.0, atol=1e-3)
    mask_o = np.isclose(delta, _DECREMENT_ONE_MAX_SENTINEL, rtol=0.0, atol=1e-3)
    mask_m = mask_z | mask_o
    vals = delta[~mask_m]
    if vals.size > 0:
        vmin, vmax = float(np.min(vals)), float(np.max(vals))
    else:
        vmin, vmax = 0.0, 1.0
    if not np.isfinite(vmin) or not np.isfinite(vmax):
        vmin, vmax = 0.0, 1.0
    if abs(vmax - vmin) < 1e-30:
        vmax = vmin + 1e-30
    norm = Normalize(vmin=vmin, vmax=vmax)
    cmap = _viridis_cmap()

    xmin_d, xmax_d = float(np.min(bx)), float(np.max(bx))
    ymin_d, ymax_d = float(np.min(fy)), float(np.max(fy))
    xr = (xmax_d - xmin_d) * 0.5 * 1.1
    yr = (ymax_d - ymin_d) * 0.5 * 1.1
    half = max(xr, yr, 0.5)
    xc = 0.5 * (xmin_d + xmax_d)
    yc = 0.5 * (ymin_d + ymax_d)
    xmin_p, xmax_p = xc - half, xc + half
    ymin_p, ymax_p = yc - half, yc + half

    pts: list[dict] = []
    for i in range(len(bx)):
        xi, yi, di = float(bx[i]), float(fy[i]), float(delta[i])
        if mask_z[i]:
            pts.append({"x": xi, "y": yi, "d": di, "k": "z"})
        elif mask_o[i]:
            pts.append({"x": xi, "y": yi, "d": di, "k": "o"})
        elif np.isclose(di, 0.0, rtol=0.0, atol=1e-12):
            pts.append({"x": xi, "y": yi, "d": di, "k": "0", "c": to_hex(cmap(norm(di)))})
        else:
            pts.append({"x": xi, "y": yi, "d": di, "k": "s", "c": to_hex(cmap(norm(di)))})

    payload = {
        "bounds": {
            "xmin": xmin_p,
            "xmax": xmax_p,
            "ymin": ymin_p,
            "ymax": ymax_p,
        },
        "points": pts,
    }
    payload_json = json.dumps(payload, ensure_ascii=False)

    fname = f"decrement_map_node{node_id}_interactive.html"
    out = plots_dir / fname

    node_esc = html_lib.escape(str(node_id))
    html_tmpl = r"""<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1"/>
  <title>Карта декремента, узел __NODE__</title>
  <style>
    body { font-family: system-ui, sans-serif; margin: 16px; background: #f5f5f5; }
    h1 { font-size: 1.1rem; }
    #chart { background: #fff; border: 1px solid #ccc; max-width: 100%; }
    #info {
      margin-top: 12px; padding: 12px; background: #fff; border: 1px solid #ccc;
      min-height: 2.5em; font-family: ui-monospace, monospace; font-size: 0.95rem;
      white-space: pre-wrap;
    }
    .hint { color: #555; font-size: 0.9rem; margin-bottom: 8px; }
  </style>
</head>
<body>
  <h1>Карта декремента (узел __NODE__) — клик по точке</h1>
  <p class="hint">Круг: δ≈0; квадрат: δ≠0; красный ▲: маркер zeroMaxs; жёлтый ▲: маркер oneMax.</p>
  <svg id="chart" width="800" height="640" viewBox="0 0 800 640" xmlns="http://www.w3.org/2000/svg">
    <rect x="0" y="0" width="800" height="600" fill="#fafafa"/>
    <g id="markers"></g>
  </svg>
  <div id="info">Нажмите на точку карты…</div>
  <script id="payload" type="application/json">__PAYLOAD__</script>
  <script>
(function() {
  const data = JSON.parse(document.getElementById('payload').textContent);
  const b = data.bounds;
  const pts = data.points;
  const marginL = 50, marginT = 30, plotW = 720, plotH = 540;
  function tx(x) {
    return marginL + (x - b.xmin) / (b.xmax - b.xmin) * plotW;
  }
  function ty(y) {
    return marginT + plotH - (y - b.ymin) / (b.ymax - b.ymin) * plotH;
  }
  const g = document.getElementById('markers');
  const info = document.getElementById('info');

  function show(p) {
    info.textContent =
      'koef1 (forward) = ' + p.x + '\n' +
      'koef2 (back)    = ' + p.y + '\n' +
      'delta            = ' + p.d;
  }

  pts.forEach(function(p) {
    const cx = tx(p.x), cy = ty(p.y);
    let el;
    if (p.k === 'z') {
      el = document.createElementNS('http://www.w3.org/2000/svg', 'polygon');
      const h = 14;
      el.setAttribute('points',
        (cx) + ',' + (cy - h) + ' ' + (cx - h * 0.9) + ',' + (cy + h * 0.55) + ' ' + (cx + h * 0.9) + ',' + (cy + h * 0.55));
      el.setAttribute('fill', 'red');
      el.setAttribute('stroke', '#600');
      el.setAttribute('stroke-width', '1');
    } else if (p.k === 'o') {
      el = document.createElementNS('http://www.w3.org/2000/svg', 'polygon');
      const h = 14;
      el.setAttribute('points',
        (cx) + ',' + (cy - h) + ' ' + (cx - h * 0.9) + ',' + (cy + h * 0.55) + ' ' + (cx + h * 0.9) + ',' + (cy + h * 0.55));
      el.setAttribute('fill', '#ffd700');
      el.setAttribute('stroke', '#8b6914');
      el.setAttribute('stroke-width', '1');
    } else if (p.k === '0') {
      el = document.createElementNS('http://www.w3.org/2000/svg', 'circle');
      el.setAttribute('cx', String(cx));
      el.setAttribute('cy', String(cy));
      el.setAttribute('r', '9');
      el.setAttribute('fill', p.c);
      el.setAttribute('stroke', '#222');
      el.setAttribute('stroke-width', '0.6');
    } else {
      el = document.createElementNS('http://www.w3.org/2000/svg', 'rect');
      const w = 16;
      el.setAttribute('x', String(cx - w/2));
      el.setAttribute('y', String(cy - w/2));
      el.setAttribute('width', String(w));
      el.setAttribute('height', String(w));
      el.setAttribute('fill', p.c);
      el.setAttribute('stroke', '#222');
      el.setAttribute('stroke-width', '0.6');
    }
    el.style.cursor = 'pointer';
    el.addEventListener('click', function(ev) {
      ev.stopPropagation();
      show(p);
    });
    g.appendChild(el);
  });
})();
  </script>
</body>
</html>
"""
    html_final = html_tmpl.replace("__NODE__", node_esc).replace("__PAYLOAD__", payload_json)

    out.write_text(html_final, encoding="utf-8")
    return fname


def get_node_ids_from_config(root_dir: Path) -> list[int]:
    """
    Читает активный конфиг (CONFIG или conf.json) и возвращает id узлов,
    для которых нужно строить графики. По умолчанию — только подвижные.
    """
    config_name = os.environ.get("CONFIG", "conf")
    config_path = root_dir / "internal" / "config" / "confs" / f"{config_name}.json"

    if not config_path.exists():
        print(f"Config not found: {config_path}, строю графики для всех graph_points*.txt")
        return []

    try:
        with config_path.open("r", encoding="utf-8") as f:
            cfg = json.load(f)
    except Exception as e:
        print(f"Не удалось прочитать конфиг {config_path}: {e}. Строю графики для всех graph_points*.txt")
        return []

    nodes = cfg.get("nodes", [])
    # Берём все узлы, включая закреплённые
    ids = [int(n["id"]) for n in nodes]
    ids = sorted(set(ids))
    return ids


def get_movable_node_ids_from_config(root_dir: Path) -> list[int]:
    config_name = os.environ.get("CONFIG", "conf")
    config_path = root_dir / "internal" / "config" / "confs" / f"{config_name}.json"
    if not config_path.exists():
        return []
    try:
        with config_path.open("r", encoding="utf-8") as f:
            cfg = json.load(f)
    except Exception:
        return []
    nodes = cfg.get("nodes", [])
    ids = [int(n["id"]) for n in nodes if not n.get("fixed", False)]
    return sorted(set(ids))


def parse_mean_delta_by_node_from_log(text: str) -> dict[int, float]:
    """Строки сводки PrintLogDecrementForAllNodes: Node id: ... delta=..."""
    out: dict[int, float] = {}
    pat = re.compile(
        r"^Node\s+(\d+):.*?delta=([+-]?(?:\d*\.)?\d+(?:[eE][+-]?\d+)?)",
        re.MULTILINE,
    )
    for m in pat.finditer(text):
        out[int(m.group(1))] = float(m.group(2))
    return out


def mean_delta_from_decrement_points_file(path: Path) -> float | None:
    _, d = load_two_column_txt(path)
    if d.size == 0:
        return None
    return float(np.mean(d))


def mean_peak_spacing_from_amplitude_file(path: Path) -> float | None:
    """Средний интервал между моментами соседних пиков амплитуды (первая колонка)."""
    t, _ = load_two_column_txt(path)
    if t.size < 2:
        return None
    dt = np.diff(t)
    positive = dt[dt > 0]
    if positive.size == 0:
        return None
    return float(np.mean(positive))


def get_integration_dt_from_config(root_dir: Path) -> float | None:
    config_name = os.environ.get("CONFIG", "conf")
    config_path = root_dir / "internal" / "config" / "confs" / f"{config_name}.json"
    if not config_path.exists():
        return None
    try:
        with config_path.open("r", encoding="utf-8") as f:
            cfg = json.load(f)
    except Exception:
        return None
    times = cfg.get("times") or {}
    dt = times.get("dt")
    if dt is None:
        return None
    try:
        v = float(dt)
    except (TypeError, ValueError):
        return None
    return v if v > 0 else None


def graph_points_t_max(path: Path) -> float | None:
    t, _ = load_two_column_txt(path)
    if t.size == 0:
        return None
    mx = float(np.max(t))
    return mx if mx > 0 else None


def node_ids_from_graph_points_glob(data_dir: Path) -> list[int]:
    ids: list[int] = []
    for p in sorted(data_dir.glob("graph_points*.txt")):
        m = re.match(r"^graph_points(\d+)$", p.stem)
        if m:
            ids.append(int(m.group(1)))
    return ids


def analytical_d_amplitude_dt(t: np.ndarray, delta_bar: float, dt_char: float, a0: float) -> np.ndarray:
    """
    dA/dt ≈ -(δ̄/Δt) A₀ exp(-δ̄ t / Δt) при экспоненциальной огибающей A(t)≈A₀ exp(-δ̄ t/Δt).
    """
    alpha = delta_bar / dt_char
    return -alpha * a0 * np.exp(-alpha * t)


def generate_plots_and_html():
    # Текущий файл: <root>/plots/plot_points.py
    plots_dir = Path(__file__).resolve().parent
    root_dir = plots_dir.parent

    data_dir = root_dir / "wolfram" / "paramsAndPoints"
    if not data_dir.exists():
        raise FileNotFoundError(f"Директория с данными не найдена: {data_dir}")

    plots_dir.mkdir(parents=True, exist_ok=True)

    if PLOT_DARK_EXPORT:
        try:
            plt.style.use("dark_background")
        except OSError:
            plt.style.use("ggplot")

    # ---------- Траектории graph_points*.txt ----------
    node_ids = get_node_ids_from_config(root_dir)

    if node_ids:
        # Берём только файлы для узлов из конфига
        graph_files = []
        for node_id in node_ids:
            path = data_dir / f"graph_points{node_id}.txt"
            if path.exists():
                graph_files.append(path)
            else:
                print(f"Файл для узла {node_id} не найден: {path}")
    else:
        # Фолбэк: все файлы
        graph_files = sorted(data_dir.glob("graph_points*.txt"))

    traj_img_name = "trajectories.png"
    amp_img_name = "amplitudes.png"
    energy_img_name = "sum_energy.png"
    dsum_energy_img_name = "dsum_energy.png"
    node_energy_img_name = "node_energy.png"
    node_d_energy_img_name = "node_d_energy.png"
    analytical_dadt_img_name = "analytical_dadt.png"
    d_amp_img_name = "d_amplitudes.png"
    dec_img_name = "decrement_deltas.png"
    decr_map_imgs = []
    energy_per_node_has_data = False
    d_energy_per_node_has_data = False
    analytical_dadt_has_data = False
    d_amp_has_data = False

    if (not PARALLEL_ONLY_DECREMENT_MAPS) and graph_files:
        plt.figure(figsize=(10, 6))
        colors = plt.cm.tab10.colors

        for idx, path in enumerate(graph_files):
            t, x = load_two_column_txt(path)
            if t.size == 0:
                continue
            label = path.stem  # например, 'graph_points0'
            plt.plot(t, x, label=label, color=colors[idx % len(colors)])

        plt.xlabel("t")
        plt.ylabel("position")
        plt.title("Trajectories from graph_points*.txt")
        plt.grid(True, alpha=0.3)
        plt.legend()
        safe_tight_layout()

        out_traj = plots_dir / traj_img_name
        plt.savefig(out_traj, dpi=200)
        plt.close()
    elif not PARALLEL_ONLY_DECREMENT_MAPS:
        print(f"Файлы graph_points*.txt не найдены в {data_dir}")

    # ---------- Амплитуды amplitude_points*.txt ----------
    # Берём именно готовые файлы амплитуд и просто отображаем точки с них.
    # Количество/набор узлов — из активного конфига (как для траекторий).
    movable_node_ids = get_movable_node_ids_from_config(root_dir)
    amp_files = []
    if movable_node_ids:
        for node_id in movable_node_ids:
            path = data_dir / f"amplitude_points{node_id}.txt"
            if path.exists():
                amp_files.append(path)
    else:
        amp_files = sorted(data_dir.glob("amplitude_points*.txt"))

    if (not PARALLEL_ONLY_DECREMENT_MAPS) and amp_files:
        plt.figure(figsize=(10, 6))
        colors = plt.cm.tab10.colors

        for idx, path in enumerate(amp_files):
            t, a = load_two_column_txt(path)
            if t.size == 0:
                continue
            label = path.stem  # например, 'amplitude_points0'
            # Показываем именно точки A_n из файла, без дополнительной обработки.
            plt.plot(t, a, marker="o", linestyle="-", label=label, color=colors[idx % len(colors)])

        plt.xlabel("t")
        plt.ylabel("|x - x_eq|")
        plt.title("Amplitudes from amplitude_points*.txt")
        plt.grid(True, alpha=0.3)
        plt.legend()
        safe_tight_layout()

        out_amp = plots_dir / amp_img_name
        plt.savefig(out_amp, dpi=200)
        plt.close()
    elif not PARALLEL_ONLY_DECREMENT_MAPS:
        print(f"Файлы amplitude_points*.txt не найдены в {data_dir}")

    # ---------- Декремент затухания по пикам decrement_points*.txt ----------
    dec_files = []
    if movable_node_ids:
        for node_id in movable_node_ids:
            path = data_dir / f"decrement_points{node_id}.txt"
            if path.exists():
                dec_files.append(path)
    else:
        dec_files = sorted(data_dir.glob("decrement_points*.txt"))

    dec_has_data = False
    if (not PARALLEL_ONLY_DECREMENT_MAPS) and dec_files:
        plt.figure(figsize=(10, 6))
        colors = plt.cm.tab10.colors

        for idx, path in enumerate(dec_files):
            t, d = load_two_column_txt(path)
            if t.size == 0:
                continue
            dec_has_data = True
            label = path.stem  # например, 'decrement_points0'
            plt.plot(t, d, marker="o", linestyle="-", label=label, color=colors[idx % len(colors)])

        plt.xlabel("t")
        plt.ylabel("delta_n")
        plt.title("Logarithmic decrement per peak pair")
        plt.grid(True, alpha=0.3)
        plt.legend()
        safe_tight_layout()

        out_dec = plots_dir / dec_img_name
        plt.savefig(out_dec, dpi=200)
        plt.close()
    elif not PARALLEL_ONLY_DECREMENT_MAPS:
        print(f"Файлы decrement_points*.txt не найдены в {data_dir}")

    # ---------- Карты декремента по аэрокоэффициентам decrementStore*.txt ----------
    # Количество карт также привязываем к активному конфигу.
    decr_store_files = []
    if movable_node_ids:
        for node_id in movable_node_ids:
            path = data_dir / f"decrementStore{node_id}.txt"
            if path.exists():
                decr_store_files.append(path)
    else:
        decr_store_files = sorted(data_dir.glob("decrementStore*.txt"))
    decr_store_has_data = False

    for path in decr_store_files:
        bx, fy, delta = load_three_column_txt(path)
        if bx.size == 0:
            continue

        decr_store_has_data = True
        # decrementStore{nodeID}.txt -> nodeID
        node_id = path.stem.replace("decrementStore", "")
        img_name = f"decrement_map_node{node_id}.png"

        mask_zero_maxs = np.isclose(delta, _DECREMENT_ZERO_MAXS_SENTINEL, rtol=0.0, atol=1e-3)
        mask_one_max = np.isclose(delta, _DECREMENT_ONE_MAX_SENTINEL, rtol=0.0, atol=1e-3)
        mask_marker = mask_zero_maxs | mask_one_max
        bx_col, fy_col, d_col = bx[~mask_marker], fy[~mask_marker], delta[~mask_marker]
        bx_red, fy_red = bx[mask_zero_maxs], fy[mask_zero_maxs]
        bx_yel, fy_yel = bx[mask_one_max], fy[mask_one_max]

        plt.figure(figsize=(8, 6))
        sc = None
        if bx_col.size > 0:
            mask_d0 = np.isclose(d_col, 0.0, rtol=0.0, atol=1e-12)
            vmin, vmax = float(np.min(d_col)), float(np.max(d_col))
            if not np.isfinite(vmin) or not np.isfinite(vmax):
                vmin, vmax = 0.0, 1.0
            if abs(vmax - vmin) < 1e-30:
                vmax = vmin + 1e-30
            norm = Normalize(vmin=vmin, vmax=vmax)
            if np.any(mask_d0):
                sc = plt.scatter(
                    bx_col[mask_d0],
                    fy_col[mask_d0],
                    c=d_col[mask_d0],
                    cmap="viridis",
                    norm=norm,
                    marker="o",
                    s=80,
                    edgecolors="k",
                    linewidths=0.25,
                )
            if np.any(~mask_d0):
                sc_sq = plt.scatter(
                    bx_col[~mask_d0],
                    fy_col[~mask_d0],
                    c=d_col[~mask_d0],
                    cmap="viridis",
                    norm=norm,
                    marker="s",
                    s=80,
                    edgecolors="k",
                    linewidths=0.25,
                )
                sc = sc_sq
        if bx_red.size > 0:
            plt.scatter(
                bx_red,
                fy_red,
                marker="^",
                c="red",
                s=110,
                edgecolors="darkred",
                linewidths=0.45,
                zorder=10,
            )
        if bx_yel.size > 0:
            plt.scatter(
                bx_yel,
                fy_yel,
                marker="^",
                c="gold",
                s=110,
                edgecolors="darkgoldenrod",
                linewidths=0.45,
                zorder=11,
            )
        plt.xlabel("koef1 (forward)")
        plt.ylabel("koef2 (back)")
        plt.title(f"Mean decrement map for node {node_id}")
        plt.grid(True, alpha=0.25)
        plt.gca().set_aspect("equal", adjustable="box")
        if sc is not None:
            cbar = plt.colorbar(sc)
            cbar.set_label("delta")
        safe_tight_layout()

        out_map = plots_dir / img_name
        plt.savefig(out_map, dpi=200)
        plt.close()

        ih = _write_decrement_map_interactive_html(plots_dir, node_id, bx, fy, delta)
        decr_map_imgs.append((node_id, img_name, ih or ""))

    if decr_store_files and not decr_store_has_data:
        print(f"Файлы decrementStore*.txt найдены, но пустые: {data_dir}")
    elif not decr_store_files:
        print(f"Файлы decrementStore*.txt не найдены в {data_dir}")

    # ---------- Суммарная энергия sumEnergyPoints.txt ----------
    energy_path = data_dir / "sumEnergyPoints.txt"
    energy_exists = energy_path.exists()

    if (not PARALLEL_ONLY_DECREMENT_MAPS) and energy_exists:
        tE, E = load_two_column_txt(energy_path)

        plt.figure(figsize=(10, 4))
        plt.plot(tE, E, label="Total energy")
        plt.xlabel("t")
        plt.ylabel("E(t)")
        plt.title("Sum of kinetic + potential energy")
        plt.grid(True, alpha=0.3)
        plt.legend()
        safe_tight_layout()

        out_energy = plots_dir / energy_img_name
        plt.savefig(out_energy, dpi=200)
        plt.close()
    elif not PARALLEL_ONLY_DECREMENT_MAPS:
        print(f"Файл с энергией не найден: {energy_path}")

    # ---------- Производная полной энергии dSumEnergyPoints.txt ----------
    dsum_energy_path = data_dir / "dSumEnergyPoints.txt"
    dsum_energy_exists = dsum_energy_path.exists()

    if (not PARALLEL_ONLY_DECREMENT_MAPS) and dsum_energy_exists:
        tdE, dE = load_two_column_txt(dsum_energy_path)

        plt.figure(figsize=(10, 4))
        plt.plot(tdE, dE, label="d/dt (K+U)")
        plt.xlabel("t")
        plt.ylabel("dE/dt")
        plt.title("Derivative of total energy")
        plt.grid(True, alpha=0.3)
        plt.legend()
        safe_tight_layout()

        out_dsum_energy = plots_dir / dsum_energy_img_name
        plt.savefig(out_dsum_energy, dpi=200)
        plt.close()
    elif not PARALLEL_ONLY_DECREMENT_MAPS:
        print(f"Файл с производной полной энергии не найден: {dsum_energy_path}")

    # ---------- Энергия по узлам energy_points*.txt (только без PARALLEL) ----------
    energy_per_node_files = []
    if movable_node_ids:
        for node_id in movable_node_ids:
            path = data_dir / f"energy_points{node_id}.txt"
            if path.exists():
                energy_per_node_files.append(path)
    else:
        energy_per_node_files = sorted(data_dir.glob("energy_points*.txt"))

    if (not PARALLEL_ONLY_DECREMENT_MAPS) and energy_per_node_files:
        plt.figure(figsize=(10, 6))
        colors = plt.cm.tab10.colors
        for idx, path in enumerate(energy_per_node_files):
            t, E = load_two_column_txt(path)
            if t.size == 0:
                continue
            energy_per_node_has_data = True
            plt.plot(t, E, label=path.stem, color=colors[idx % len(colors)])
        if energy_per_node_has_data:
            plt.xlabel("t")
            plt.ylabel("E(t)")
            plt.title("Energy per node (energy_points*.txt)")
            plt.grid(True, alpha=0.3)
            plt.legend()
            safe_tight_layout()
            plt.savefig(plots_dir / node_energy_img_name, dpi=200)
        plt.close()
    elif not PARALLEL_ONLY_DECREMENT_MAPS and not energy_per_node_files:
        print(f"Файлы energy_points*.txt не найдены в {data_dir}")

    # ---------- Производная энергии по узлам d_energy_points*.txt (только без PARALLEL) ----------
    d_energy_per_node_files = []
    if movable_node_ids:
        for node_id in movable_node_ids:
            path = data_dir / f"d_energy_points{node_id}.txt"
            if path.exists():
                d_energy_per_node_files.append(path)
    else:
        d_energy_per_node_files = sorted(data_dir.glob("d_energy_points*.txt"))

    if (not PARALLEL_ONLY_DECREMENT_MAPS) and d_energy_per_node_files:
        plt.figure(figsize=(10, 6))
        colors = plt.cm.tab10.colors
        for idx, path in enumerate(d_energy_per_node_files):
            t, d_e = load_two_column_txt(path)
            if t.size == 0:
                continue
            d_energy_per_node_has_data = True
            plt.plot(t, d_e, label=path.stem, color=colors[idx % len(colors)])
        if d_energy_per_node_has_data:
            plt.xlabel("t")
            plt.ylabel("dE/dt")
            plt.title("dE/dt per node (d_energy_points*.txt)")
            plt.grid(True, alpha=0.3)
            plt.legend()
            safe_tight_layout()
            plt.savefig(plots_dir / node_d_energy_img_name, dpi=200)
        plt.close()
    elif not PARALLEL_ONLY_DECREMENT_MAPS and not d_energy_per_node_files:
        print(f"Файлы d_energy_points*.txt не найдены в {data_dir}")

    # ---------- Аналитическая dA/dt от среднего δ (только без PARALLEL) ----------
    if not PARALLEL_ONLY_DECREMENT_MAPS:
        logs_an_path = data_dir / "decrement_details.log"
        log_an_text = ""
        if logs_an_path.exists():
            try:
                log_an_text = logs_an_path.read_text(encoding="utf-8")
            except Exception:
                log_an_text = ""
        delta_by_node = parse_mean_delta_by_node_from_log(log_an_text)

        raw_a0 = os.environ.get("A0", "1").strip()
        try:
            a0_plot = float(raw_a0)
        except ValueError:
            a0_plot = 1.0

        dt_src = os.environ.get("ANALYTICAL_DT_SOURCE", "peak").strip().lower()
        integr_dt = get_integration_dt_from_config(root_dir)

        nodes_for_analytical = (
            movable_node_ids if movable_node_ids else node_ids_from_graph_points_glob(data_dir)
        )

        plt.figure(figsize=(10, 6))
        colors = plt.cm.tab10.colors
        plot_idx = 0
        for node_id in nodes_for_analytical:
            dbar = delta_by_node.get(node_id)
            if dbar is None:
                dp_path = data_dir / f"decrement_points{node_id}.txt"
                if dp_path.exists():
                    dbar = mean_delta_from_decrement_points_file(dp_path)
            if dbar is None or dbar <= 0:
                continue

            dt_char: float | None = None
            if dt_src in ("integrator", "dt", "config"):
                dt_char = integr_dt
            else:
                amp_p = data_dir / f"amplitude_points{node_id}.txt"
                if amp_p.exists():
                    dt_char = mean_peak_spacing_from_amplitude_file(amp_p)
                if (dt_char is None or dt_char <= 0) and integr_dt is not None:
                    dt_char = integr_dt

            if dt_char is None or dt_char <= 0:
                print(f"Узел {node_id}: не задан положительный Δt для аналитики dA/dt, пропуск.")
                continue

            gp = data_dir / f"graph_points{node_id}.txt"
            if not gp.exists():
                continue
            t_max = graph_points_t_max(gp)
            if t_max is None:
                continue

            n_pts = max(80, min(4000, int(t_max / max(dt_char * 0.05, 1e-9))))
            t_fine = np.linspace(0.0, t_max, n_pts)
            y = analytical_d_amplitude_dt(t_fine, dbar, dt_char, a0_plot)
            lim = _PLOT_COL_ABS_MAX
            m = np.isfinite(t_fine) & np.isfinite(y) & (np.abs(y) <= lim)
            if not np.any(m):
                continue
            lbl = f"node {node_id} (δ̄={dbar:.4g}, Δt={dt_char:.4g})"
            plt.plot(t_fine[m], y[m], label=lbl, color=colors[plot_idx % len(colors)])
            analytical_dadt_has_data = True
            plot_idx += 1

        if analytical_dadt_has_data:
            plt.xlabel("t")
            plt.ylabel("dA/dt")
            src_note = "Δt: интервал между пиками" if dt_src not in ("integrator", "dt", "config") else "Δt: times.dt из конфига"
            plt.title(
                r"Аналитика $\frac{dA}{dt}\approx -\frac{\bar\delta}{\Delta t} A_0 \exp(-\bar\delta t/\Delta t)$"
                + f", $A_0$={a0_plot:g}; {src_note}"
            )
            plt.grid(True, alpha=0.3)
            plt.legend(fontsize=8)
            safe_tight_layout()
            plt.savefig(plots_dir / analytical_dadt_img_name, dpi=200)
        plt.close()

        # ---------- dA/dt по узлам из d_amplitude_points*.txt ----------
        d_amp_files = []
        if movable_node_ids:
            for node_id in movable_node_ids:
                p = data_dir / f"d_amplitude_points{node_id}.txt"
                if p.exists():
                    d_amp_files.append(p)
        else:
            d_amp_files = sorted(data_dir.glob("d_amplitude_points*.txt"))

        if d_amp_files:
            plt.figure(figsize=(10, 6))
            colors = plt.cm.tab10.colors
            for idx, path in enumerate(d_amp_files):
                t, dA = load_two_column_txt(path)
                if t.size == 0:
                    continue
                d_amp_has_data = True
                plt.plot(t, dA, marker="o", linestyle="-", label=path.stem, color=colors[idx % len(colors)])

            if d_amp_has_data:
                plt.xlabel("t")
                plt.ylabel("dA/dt")
                plt.title("dA/dt per node (d_amplitude_points*.txt)")
                plt.grid(True, alpha=0.3)
                plt.legend()
                safe_tight_layout()
                plt.savefig(plots_dir / d_amp_img_name, dpi=200)
            plt.close()
        else:
            print(f"Файлы d_amplitude_points*.txt не найдены в {data_dir}")

    # ---------- Генерация HTML ----------
    html_path = plots_dir / "index.html"
    logs_path = data_dir / "decrement_details.log"
    logs_text = ""
    if logs_path.exists():
        try:
            logs_text = logs_path.read_text(encoding="utf-8")
        except Exception:
            logs_text = ""
    logs_html = "<p>Логи отсутствуют.</p>" if not logs_text.strip() else f"<pre>{html_lib.escape(logs_text)}</pre>"

    # Сводка по нодам: берём строки вида
    # "Node 0: xEq=..., skipFirst=..., usedPairs=..., delta=..., decay=...%"
    node_summary_lines = []
    for line in logs_text.splitlines():
        stripped = line.strip()
        if stripped.startswith("Node ") and "xEq=" in stripped and "delta=" in stripped and "decay=" in stripped:
            node_summary_lines.append(stripped)
    if node_summary_lines:
        node_summary_html = "<ul>" + "".join(f"<li><code>{html_lib.escape(line)}</code></li>" for line in node_summary_lines) + "</ul>"
    else:
        node_summary_html = "<p>Сводка по нодам отсутствует.</p>"

    formulas_html = """
<div class="formula-block" style="line-height:1.8">
  <p><b>Логарифмический декремент по соседним пикам:</b></p>
  <math display="block">
    <msub><mi>&delta;</mi><mi>n</mi></msub>
    <mo>=</mo>
    <mi>ln</mi>
    <mfrac>
      <msub><mi>A</mi><mi>n</mi></msub>
      <msub><mi>A</mi><mrow><mi>n</mi><mo>+</mo><mn>1</mn></mrow></msub>
    </mfrac>
  </math>

  <p><b>Средний декремент по интервалу:</b></p>
  <math display="block">
    <mi>&delta;</mi>
    <mo>=</mo>
    <mfrac><mn>1</mn><mi>N</mi></mfrac>
    <munderover>
      <mo>&Sigma;</mo>
      <mrow><mi>n</mi><mo>=</mo><mn>1</mn></mrow>
      <mi>N</mi>
    </munderover>
    <msub><mi>&delta;</mi><mi>n</mi></msub>
  </math>

  <p><b>Связь с процентным уменьшением амплитуды за один пик-период:</b></p>
  <math display="block">
    <mi>decay</mi><mo>(</mo><mo>%</mo><mo>)</mo>
    <mo>=</mo>
    <mo>(</mo><mn>1</mn><mo>-</mo>
    <msup><mi>e</mi><mrow><mo>-</mo><mi>&delta;</mi></mrow></msup>
    <mo>)</mo>
    <mo>&middot;</mo><mn>100</mn><mo>%</mo>
  </math>

  <p><b>Амплитуда пика:</b></p>
  <math display="block">
    <msub><mi>A</mi><mi>n</mi></msub>
    <mo>=</mo>
    <mo>|</mo>
    <mi>x</mi><mo>(</mo><msub><mi>t</mi><mi>n</mi></msub><mo>)</mo>
    <mo>-</mo>
    <msub><mi>x</mi><mrow><mi>e</mi><mi>q</mi></mrow></msub>
    <mo>|</mo>
  </math>
  <p class="formula-note"><b>Где</b> <code>t<sub>n</sub></code> — момент времени, в который достигается n-й амплитудный пик (локальный максимум амплитуды).</p>
</div>
"""

    plot_dark_attr = "1" if PLOT_DARK_EXPORT else "0"
    html_content = f"""<!DOCTYPE html>
<html lang="ru" data-export-dark="{plot_dark_attr}">
<head>
  <meta charset="UTF-8">
  <title>Графики траекторий и энергии</title>
  <script>
    (function () {{
      try {{
        var t = localStorage.getItem("university_plots_theme");
        if (t === "dark" || t === "light")
          document.documentElement.setAttribute("data-theme", t);
      }} catch (e) {{}}
    }})();
  </script>
  <style>
    html {{
      color-scheme: light dark;
    }}
    body {{
      font-family: sans-serif;
      margin: 20px;
      padding-top: 48px;
      background: #fafafa;
      color: #111;
    }}
    html[data-theme="dark"] body {{
      background: #1a1a1e;
      color: #e8e8ed;
    }}
    h1, h2 {{
      font-weight: 600;
    }}
    html[data-theme="dark"] h1,
    html[data-theme="dark"] h2 {{
      color: #f0f0f5;
    }}
    .theme-bar {{
      position: fixed;
      top: 12px;
      right: 16px;
      z-index: 1000;
    }}
    #theme-toggle {{
      cursor: pointer;
      padding: 8px 14px;
      font-size: 0.95rem;
      border-radius: 8px;
      border: 1px solid #bbb;
      background: #fff;
      color: #222;
      box-shadow: 0 1px 3px rgba(0,0,0,0.12);
    }}
    html[data-theme="dark"] #theme-toggle {{
      border-color: #555;
      background: #2d2d33;
      color: #e8e8ed;
      box-shadow: 0 1px 3px rgba(0,0,0,0.4);
    }}
    #theme-toggle:hover {{
      filter: brightness(0.97);
    }}
    html[data-theme="dark"] #theme-toggle:hover {{
      filter: brightness(1.08);
    }}
    img {{
      max-width: 100%;
      height: auto;
      border: 1px solid #ccc;
      background: #fff;
      padding: 4px;
      box-shadow: 0 2px 4px rgba(0,0,0,0.1);
      transition: filter 0.25s ease;
    }}
    html[data-theme="dark"] img {{
      border-color: #444;
      background: #121215;
      box-shadow: 0 2px 8px rgba(0,0,0,0.45);
    }}
    /* Затемняем «белые» PNG только если они собраны без PLOT_DARK */
    html[data-theme="dark"]:not([data-export-dark="1"]) img {{
      filter: brightness(0.68) contrast(1.14) saturate(0.45);
    }}
    /* Чуть приглушаем уже тёмные PNG при переключателе */
    html[data-theme="dark"][data-export-dark="1"] img {{
      filter: brightness(0.88) saturate(0.82);
    }}
    .block {{
      margin-bottom: 32px;
    }}
    pre {{
      background: #fff;
      border: 1px solid #ddd;
      padding: 12px;
      overflow-x: auto;
      border-radius: 6px;
    }}
    html[data-theme="dark"] pre {{
      background: #121215;
      border-color: #444;
      color: #ddd;
    }}
    code {{
      background: #eee;
      padding: 1px 5px;
      border-radius: 4px;
    }}
    html[data-theme="dark"] code {{
      background: #2d2d33;
      color: #e0e0e8;
    }}
    a {{
      color: #0b57d0;
    }}
    html[data-theme="dark"] a {{
      color: #8ab4ff;
    }}
    ul li {{
      margin-bottom: 0.35em;
    }}
    .formula-block math {{
      font-size: 1.55em;
    }}
    .formula-note {{
      margin-top: 8px;
      font-size: 1.08em;
      color: #202020;
    }}
    html[data-theme="dark"] .formula-note {{
      color: #c8c8d0;
    }}
  </style>
</head>
<body>
  <div class="theme-bar">
    <button type="button" id="theme-toggle" aria-label="Переключить тему оформления">Тёмная тема</button>
  </div>
  <h1>Графики из wolfram/paramsAndPoints</h1>

  {"" if PARALLEL_ONLY_DECREMENT_MAPS else f'''
  <div class="block">
    <h2>Траектории узлов (graph_points*.txt)</h2>
    {"<p>Файлы не найдены.</p>" if not graph_files else f'<img src="{traj_img_name}" alt="Trajectories">'}
  </div>

  <div class="block">
    <h2>Амплитуды узлов (amplitude_points*.txt)</h2>
    {"<p>Файлы не найдены.</p>" if not amp_files else f'<img src="{amp_img_name}" alt="Amplitudes">'}
  </div>

  <div class="block">
    <h2>Суммарная энергия (sumEnergyPoints.txt)</h2>
    {"<p>Файл не найден.</p>" if not energy_exists else f'<img src="{energy_img_name}" alt="Total energy">'}
  </div>

  <div class="block">
    <h2>Производная полной энергии (dSumEnergyPoints.txt)</h2>
    {"<p>Файл не найден.</p>" if not dsum_energy_exists else f'<img src="{dsum_energy_img_name}" alt="dE/dt">'}
  </div>

  <div class="block">
    <h2>Энергия по узлам (energy_points*.txt)</h2>
    {"<p>Файлы не найдены или нет данных.</p>" if not energy_per_node_has_data else f'<img src="{node_energy_img_name}" alt="Energy per node">'}
  </div>

  <div class="block">
    <h2>Производная энергии по узлам (d_energy_points*.txt)</h2>
    {"<p>Файлы не найдены или нет данных.</p>" if not d_energy_per_node_has_data else f'<img src="{node_d_energy_img_name}" alt="dE/dt per node">'}
  </div>

  <div class="block">
    <h2>Аналитическая производная амплитуды (по среднему δ)</h2>
    {"<p>Нет кривых: нужны graph_points*, положительный δ (decrement_details.log или decrement_points*) и Δt (пики или times.dt).</p>" if not analytical_dadt_has_data else f'<img src="{analytical_dadt_img_name}" alt="Analytical dA/dt">'}
  </div>

  <div class="block">
    <h2>Производная амплитуды по узлам (d_amplitude_points*.txt)</h2>
    {"<p>Файлы не найдены или нет данных.</p>" if not d_amp_has_data else f'<img src="{d_amp_img_name}" alt="dA/dt per node">'}
  </div>
  '''}

  {"" if PARALLEL_ONLY_DECREMENT_MAPS else f'''
  <div class="block">
    <h2>Сводка декремента по узлам</h2>
    {node_summary_html}
  </div>

  <div class="block">
    <h2>Формулы расчёта декремента</h2>
    {formulas_html}
  </div>

  <div class="block">
    <h2>Декремент затухания по парам пиков (decrement_points*.txt)</h2>
    {"<p>Файлы не найдены или пусты.</p>" if not dec_has_data else f'<img src="{dec_img_name}" alt="Decrement deltas">'}
  </div>
  '''}

  <div class="block">
    <h2>Карты декремента по аэрокоэффициентам (decrementStore*.txt)</h2>
    {"<p>Файлы не найдены или пусты.</p>" if not decr_map_imgs else "".join(
        f'<h3>Узел {html_lib.escape(str(node_id))}</h3>'
        + (
            f'<p><a href="{html_lib.escape(ih)}" target="_blank" rel="noopener">Интерактивная карта (клик по точке)</a></p>'
            if ih
            else ""
        )
        + f'<img src="{html_lib.escape(img_name)}" alt="Decrement map node {html_lib.escape(str(node_id))}"><br><br>'
        for node_id, img_name, ih in decr_map_imgs
    )}
  </div>

  {"" if PARALLEL_ONLY_DECREMENT_MAPS else f'''
  <div class="block">
    <h2>Логи расчёта декремента</h2>
    {logs_html}
  </div>

  <div class="block">
    <h2>Шпаргалка (декремент)</h2>
    <p><b>Амплитуда n-го пика:</b></p>
    <math display="block">
      <msub><mi>A</mi><mi>n</mi></msub>
      <mo>=</mo>
      <mo>|</mo>
      <mi>x</mi><mo>(</mo><msub><mi>t</mi><mi>n</mi></msub><mo>)</mo>
      <mo>-</mo>
      <msub><mi>x</mi><mrow><mi>e</mi><mi>q</mi></mrow></msub>
      <mo>|</mo>
    </math>

    <p><b>Декремент по соседним пикам:</b></p>
    <math display="block">
      <msub><mi>&delta;</mi><mi>n</mi></msub>
      <mo>=</mo>
      <mi>ln</mi>
      <mfrac>
        <msub><mi>A</mi><mi>n</mi></msub>
        <msub><mi>A</mi><mrow><mi>n</mi><mo>+</mo><mn>1</mn></mrow></msub>
      </mfrac>
    </math>

    <p><b>Средний декремент по интервалу:</b></p>
    <math display="block">
      <mi>&delta;</mi>
      <mo>=</mo>
      <mfrac><mn>1</mn><mi>N</mi></mfrac>
      <munderover>
        <mo>&Sigma;</mo>
        <mrow><mi>n</mi><mo>=</mo><mn>1</mn></mrow>
        <mi>N</mi>
      </munderover>
      <msub><mi>&delta;</mi><mi>n</mi></msub>
    </math>

    <p><b>Уменьшение амплитуды за один пик-период:</b></p>
    <math display="block">
      <mi>decay</mi><mo>(</mo><mo>%</mo><mo>)</mo>
      <mo>=</mo>
      <mo>(</mo><mn>1</mn><mo>-</mo>
      <msup><mi>e</mi><mrow><mo>-</mo><mi>&delta;</mi></mrow></msup>
      <mo>)</mo>
      <mo>&middot;</mo><mn>100</mn><mo>%</mo>
    </math>

    <p><b>Связь с коэффициентом затухания (underdamped):</b></p>
    <math display="block">
      <mi>&zeta;</mi>
      <mo>=</mo>
      <mfrac>
        <mi>&delta;</mi>
        <msqrt>
          <mrow><mn>4</mn><msup><mi>&pi;</mi><mn>2</mn></msup><mo>+</mo><msup><mi>&delta;</mi><mn>2</mn></msup></mrow>
        </msqrt>
      </mfrac>
    </math>
    <math display="block">
      <mi>&delta;</mi>
      <mo>=</mo>
      <mfrac>
        <mrow><mn>2</mn><mi>&pi;</mi><mi>&zeta;</mi></mrow>
        <msqrt><mrow><mn>1</mn><mo>-</mo><msup><mi>&zeta;</mi><mn>2</mn></msup></mrow></msqrt>
      </mfrac>
    </math>

    <p class="formula-note">Здесь <code>t<sub>n</sub></code> — момент времени, в который достигается n-й амплитудный пик.</p>
  </div>
  '''}

  <div class="block">
    <h2>Источники</h2>
    <ul>
      <li><a href="https://www.pearson.com/en-us/subject-catalog/p/mechanical-vibrations/P200000003477/9780137686078" target="_blank" rel="noopener noreferrer">S. S. Rao — Mechanical Vibrations</a></li>
      <li><a href="https://www.pearson.com/en-us/subject-catalog/p/theory-of-vibration-with-applications/P200000003392/9780136512828" target="_blank" rel="noopener noreferrer">W. T. Thomson, M. D. Dahleh, C. Padmanabhan — Theory of Vibration with Applications</a></li>
      <li><a href="https://store.doverpublications.com/products/9780486647850" target="_blank" rel="noopener noreferrer">J. P. Den Hartog — Mechanical Vibrations</a></li>
      <li><a href="https://www.pearson.com/en-us/subject-catalog/p/engineering-vibration/P200000003365/9780132871691" target="_blank" rel="noopener noreferrer">D. J. Inman — Engineering Vibration</a></li>
      <li><a href="https://www.mheducation.com/highered/product/fundamentals-vibrations-meirovitch/M9781577667429.html" target="_blank" rel="noopener noreferrer">L. Meirovitch — Fundamentals of Vibrations</a></li>
    </ul>
  </div>
  <script>
    (function () {{
      var KEY = "university_plots_theme";
      var root = document.documentElement;
      var btn = document.getElementById("theme-toggle");
      function getTheme() {{
        var a = root.getAttribute("data-theme");
        return a === "dark" ? "dark" : "light";
      }}
      function apply(theme) {{
        root.setAttribute("data-theme", theme);
        try {{
          localStorage.setItem(KEY, theme);
        }} catch (e) {{}}
        btn.textContent = theme === "dark" ? "Светлая тема" : "Тёмная тема";
        btn.setAttribute("aria-pressed", theme === "dark" ? "true" : "false");
      }}
      btn.addEventListener("click", function () {{
        apply(getTheme() === "dark" ? "light" : "dark");
      }});
      apply(getTheme());
    }})();
  </script>
</body>
</html>
"""

    html_path.write_text(html_content, encoding="utf-8")


if __name__ == "__main__":
    generate_plots_and_html()

