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
  plots/decrement_map_node*.png, plots/decrement_map_node*_interactive.html — только при PARALLEL=true
  plots/frequency_map_node*.png, plots/frequency_map_node*_interactive.html — только при PARALLEL=true

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
# true: карты декремента (decrementStore) и частоты (freqStore): PNG + интерактивный HTML.
# false: полный набор графиков без карт по аэрокоэффициентам.
PARALLEL_ONLY_DECREMENT_MAPS = env_bool("PARALLEL", default=False)

# При генерации PNG: тёмный фон и светлые оси (export PLOT_DARK=true).
PLOT_DARK_EXPORT = env_bool("PLOT_DARK", default=False)


# Верхняя граница по модулю для столбцов из txt: отсекаем огромные конечные числа, из‑за которых
# matplotlib ломается на tight_layout (tick locator / arange).
_PLOT_COL_ABS_MAX = 1e100

# Совпадают с zeroMaxs и oneMax в internal/numMethods/utils/damping_decrement.go.
_AERO_ZERO_MAXS_SENTINEL = 123456.123456
_AERO_ONE_MAX_SENTINEL = 1232323.1232323
_DECREMENT_ZERO_MAXS_SENTINEL = _AERO_ZERO_MAXS_SENTINEL
_DECREMENT_ONE_MAX_SENTINEL = _AERO_ONE_MAX_SENTINEL


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


def _aero_map_sentinel_masks(values: np.ndarray) -> tuple[np.ndarray, np.ndarray, np.ndarray]:
    mask_z = np.isclose(values, _AERO_ZERO_MAXS_SENTINEL, rtol=0.0, atol=1e-3)
    mask_o = np.isclose(values, _AERO_ONE_MAX_SENTINEL, rtol=0.0, atol=1e-3)
    return mask_z, mask_o, mask_z | mask_o


def _aero_coef_key(x: float, y: float) -> tuple[float, float]:
    return (round(float(x), 6), round(float(y), 6))


def _build_aero_coef_lookup(
    bx: np.ndarray,
    fy: np.ndarray,
    values: np.ndarray,
) -> dict[tuple[float, float], float]:
    lookup: dict[tuple[float, float], float] = {}
    for i in range(len(bx)):
        lookup[_aero_coef_key(bx[i], fy[i])] = float(values[i])
    return lookup


def _lookup_aero_coef_values(
    lookup: dict[tuple[float, float], float],
    bx: np.ndarray,
    fy: np.ndarray,
) -> np.ndarray:
    out = np.full(len(bx), np.nan, dtype=float)
    for i in range(len(bx)):
        key = _aero_coef_key(bx[i], fy[i])
        if key in lookup:
            out[i] = lookup[key]
    return out


def load_phase_diff_lookup(path: Path) -> dict[tuple[float, float], dict]:
    if not path.exists():
        return {}
    try:
        with path.open("r", encoding="utf-8") as f:
            records = json.load(f)
    except (OSError, json.JSONDecodeError) as ex:
        print(f"Не удалось прочитать {path}: {ex}")
        return {}
    if not isinstance(records, list):
        return {}
    out: dict[tuple[float, float], dict] = {}
    for rec in records:
        if not isinstance(rec, dict):
            continue
        try:
            key = _aero_coef_key(rec["koef1"], rec["koef2"])
        except (KeyError, TypeError, ValueError):
            continue
        out[key] = rec
    return out


def equilibrium_for_node_from_config(cfg: dict, node_id: int) -> float | None:
    nodes = {int(n["id"]): n for n in cfg.get("nodes", [])}
    if node_id not in nodes:
        return None
    fixed = [n for n in cfg.get("nodes", []) if n.get("fixed")]
    if not fixed:
        return None
    for fixed_node in fixed:
        fid = int(fixed_node["id"])
        fpos = float(fixed_node.get("position", 0.0))
        for e in cfg.get("edges", []):
            if int(e["from"]) == node_id and int(e["to"]) == fid:
                return fpos - float(e.get("rest", 0.0))
        for e in cfg.get("edges", []):
            if int(e["from"]) == fid and int(e["to"]) == node_id:
                return fpos + float(e.get("rest", 0.0))
    return float(nodes[node_id].get("position", 0.0))


def interblade_phase_diff_deg_numpy(
    t: np.ndarray,
    xa: np.ndarray,
    xb: np.ndarray,
    x_eq_a: float,
    x_eq_b: float,
) -> tuple[np.ndarray, np.ndarray]:
    if t.size < 5:
        return np.array([]), np.array([])
    a = xa - x_eq_a
    b = xb - x_eq_b
    va = np.gradient(a, t)
    vb = np.gradient(b, t)
    amp_a = np.hypot(a, va)
    amp_b = np.hypot(b, vb)
    mask = (amp_a > 1e-10) & (amp_b > 1e-10)
    pa = np.arctan2(-va, a)
    pb = np.arctan2(-vb, b)
    diff = np.degrees(pa - pb)
    diff = np.unwrap(np.radians(diff)) * (180.0 / np.pi)
    return t[mask], diff[mask]


def _aero_paired_scalar(v: float) -> float | None:
    if not np.isfinite(v):
        return None
    if np.isclose(v, _AERO_ZERO_MAXS_SENTINEL, rtol=0.0, atol=1e-3):
        return None
    if np.isclose(v, _AERO_ONE_MAX_SENTINEL, rtol=0.0, atol=1e-3):
        return None
    return float(v)


def _write_aero_map_interactive_html(
    plots_dir: Path,
    node_id: str,
    bx: np.ndarray,
    fy: np.ndarray,
    values: np.ndarray,
    *,
    file_stem: str,
    page_title: str,
    hint_html: str,
    value_label: str,
    distinguish_zero: bool,
    koef1_label: str = "koef1 (forward)",
    koef2_label: str = "koef2 (back)",
    labels_use_html: bool = False,
    map_metric: str | None = None,
    paired_freq: np.ndarray | None = None,
    paired_delta: np.ndarray | None = None,
    phase_lookup: dict[tuple[float, float], dict] | None = None,
) -> str | None:
    """
    Автономный HTML+SVG+JS: клик по точке показывает koef1, koef2 и значение карты.
    Маркеры и цвета согласованы со статической картой (круг/квадрат/треугольники).
    """
    if bx.size == 0:
        return None

    mask_z, mask_o, mask_m = _aero_map_sentinel_masks(values)
    vals = values[~mask_m]
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
        xi, yi, vi = float(bx[i]), float(fy[i]), float(values[i])
        delta_v: float | None = None
        freq_v: float | None = None
        if map_metric == "delta":
            if not mask_z[i] and not mask_o[i]:
                delta_v = _aero_paired_scalar(vi)
            if paired_freq is not None and i < len(paired_freq):
                freq_v = _aero_paired_scalar(float(paired_freq[i]))
        elif map_metric == "frequency":
            if not mask_z[i] and not mask_o[i]:
                freq_v = _aero_paired_scalar(vi)
            if paired_delta is not None and i < len(paired_delta):
                delta_v = _aero_paired_scalar(float(paired_delta[i]))
        if mask_z[i]:
            pt: dict = {"x": xi, "y": yi, "d": vi, "k": "z"}
        elif mask_o[i]:
            pt = {"x": xi, "y": yi, "d": vi, "k": "o"}
        elif distinguish_zero and np.isclose(vi, 0.0, rtol=0.0, atol=1e-12):
            pt = {"x": xi, "y": yi, "d": vi, "k": "0", "c": to_hex(cmap(norm(vi)))}
        else:
            pt = {"x": xi, "y": yi, "d": vi, "k": "s", "c": to_hex(cmap(norm(vi)))}
        if delta_v is not None:
            pt["delta"] = delta_v
        if freq_v is not None:
            pt["freq"] = freq_v
        if phase_lookup is not None:
            phase_rec = phase_lookup.get(_aero_coef_key(xi, yi))
            if phase_rec:
                nb = int(phase_rec.get("neighborId", -1))
                pt["phaseLabel"] = f"φ({node_id})−φ({nb})"
                pt["phaseMeanDeg"] = float(phase_rec.get("meanDeg", 0.0))
                pt["phaseStdDeg"] = float(phase_rec.get("stdDeg", 0.0))
                pt["phaseT"] = phase_rec.get("t") or []
                pt["phaseDeg"] = phase_rec.get("phaseDeg") or []
        pts.append(pt)

    show_osc_metrics = map_metric in ("delta", "frequency")

    payload = {
        "bounds": {
            "xmin": xmin_p,
            "xmax": xmax_p,
            "ymin": ymin_p,
            "ymax": ymax_p,
        },
        "valueLabel": value_label,
        "koef1Label": koef1_label,
        "koef2Label": koef2_label,
        "labelsUseHtml": labels_use_html,
        "showOscMetrics": show_osc_metrics,
        "mapMetric": map_metric,
        "nodeId": str(node_id),
        "points": pts,
    }
    payload_json = json.dumps(payload, ensure_ascii=False)

    fname = f"{file_stem}_node{node_id}_interactive.html"
    out = plots_dir / fname

    node_esc = html_lib.escape(str(node_id))
    page_title_esc = html_lib.escape(page_title)
    hint_esc = hint_html
    html_tmpl = r"""<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1"/>
  <title>__PAGE_TITLE__</title>
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
    #phasePanel {
      margin-top: 12px; padding: 12px; background: #fff; border: 1px solid #ccc;
      display: none;
    }
    #phasePanel h2 { font-size: 1rem; margin: 0 0 8px; }
    #phaseCaption { margin: 0 0 4px; }
    #phaseStats { margin: 0 0 10px; color: #444; font-size: 0.9rem; }
    #phaseChart { display: block; max-width: 100%; height: auto; }
  </style>
</head>
<body>
  <h1>__PAGE_TITLE__ (узел __NODE__) — клик по точке</h1>
  <p class="hint">__HINT__</p>
  <svg id="chart" width="800" height="640" viewBox="0 0 800 640" xmlns="http://www.w3.org/2000/svg">
    <rect x="0" y="0" width="800" height="600" fill="#fafafa"/>
    <g id="markers"></g>
  </svg>
  <div id="info">Нажмите на точку карты…</div>
  <div id="phasePanel">
    <h2>Межлопаточная фаза Δφ(t) для этой лопатки</h2>
    <p id="phaseCaption" class="hint">Кликните квадрат на карте.</p>
    <p id="phaseStats" class="hint"></p>
    <svg id="phaseChart" width="920" height="400" viewBox="0 0 920 400" xmlns="http://www.w3.org/2000/svg"></svg>
  </div>
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
  const phasePanel = document.getElementById('phasePanel');
  const phaseChart = document.getElementById('phaseChart');
  const phaseCaption = document.getElementById('phaseCaption');
  const phaseStats = document.getElementById('phaseStats');
  const svgNs = 'http://www.w3.org/2000/svg';

  const valueLabel = data.valueLabel || 'value';
  const koef1Label = data.koef1Label || 'koef1 (forward)';
  const koef2Label = data.koef2Label || 'koef2 (back)';
  function esc(s) {
    return String(s)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;');
  }
  function niceStep(range, ticks) {
    if (!isFinite(range) || range <= 0) return 1;
    const rough = range / ticks;
    const pow = Math.pow(10, Math.floor(Math.log10(rough)));
    const norm = rough / pow;
    let step = 10;
    if (norm <= 1.5) step = 1;
    else if (norm <= 3) step = 2;
    else if (norm <= 7) step = 5;
    return step * pow;
  }
  function formatTick(v, isTime) {
    if (!isFinite(v)) return '';
    const av = Math.abs(v);
    if (isTime) {
      if (av >= 100) return v.toFixed(0);
      if (av >= 10) return v.toFixed(1);
      return v.toFixed(2);
    }
    if (av >= 1000) return v.toFixed(0);
    if (av >= 100) return v.toFixed(1);
    return v.toFixed(2);
  }
  function svgText(x, y, text, anchor, size, rotate) {
    const el = document.createElementNS(svgNs, 'text');
    el.setAttribute('x', String(x));
    el.setAttribute('y', String(y));
    el.setAttribute('font-size', String(size || 11));
    el.setAttribute('fill', '#222');
    if (anchor) el.setAttribute('text-anchor', anchor);
    if (rotate) el.setAttribute('transform', rotate);
    el.textContent = text;
    return el;
  }
  function drawPhaseChart(p) {
    while (phaseChart.firstChild) {
      phaseChart.removeChild(phaseChart.firstChild);
    }
    if (!p.phaseT || !p.phaseDeg || p.phaseT.length < 2) {
      phasePanel.style.display = 'none';
      return;
    }
    phasePanel.style.display = 'block';
    const label = p.phaseLabel || ('φ(' + data.nodeId + ')−φ(?)');
    let cap = 'k\u207A=' + p.x + ', k\u207B=' + p.y;
    if (p.delta != null) cap += ', Λ=' + Number(p.delta).toFixed(6);
    if (p.freq != null) cap += ', f=' + Number(p.freq).toFixed(6);
    cap += ' — ' + label;
    phaseCaption.textContent = cap;
    if (p.phaseMeanDeg != null && p.phaseStdDeg != null) {
      phaseStats.textContent =
        'среднее Δφ за прогон = ' + Number(p.phaseMeanDeg).toFixed(2) +
        '°, СКО Δφ = ' + Number(p.phaseStdDeg).toFixed(2) +
        '° (разброс фазы во времени; это не логарифмический декремент Λ)';
    } else {
      phaseStats.textContent = '';
    }
    const W = 920, H = 400, mL = 72, mR = 24, mT = 44, mB = 56;
    const pW = W - mL - mR, pH = H - mT - mB;
    const tMin = Math.min.apply(null, p.phaseT);
    const tMax = Math.max.apply(null, p.phaseT);
    let yMin = Math.min.apply(null, p.phaseDeg);
    let yMax = Math.max.apply(null, p.phaseDeg);
    if (Math.abs(yMax - yMin) < 1e-9) {
      yMin -= 5.0;
      yMax += 5.0;
    } else {
      const pad = 0.06 * (yMax - yMin);
      yMin -= pad;
      yMax += pad;
    }
    const txp = function(t) {
      return mL + (t - tMin) / (tMax - tMin) * pW;
    };
    const typ = function(y) {
      return mT + pH - (y - yMin) / (yMax - yMin) * pH;
    };
    const bg = document.createElementNS(svgNs, 'rect');
    bg.setAttribute('x', '0');
    bg.setAttribute('y', '0');
    bg.setAttribute('width', String(W));
    bg.setAttribute('height', String(H));
    bg.setAttribute('fill', '#fafafa');
    phaseChart.appendChild(bg);
    const plotBg = document.createElementNS(svgNs, 'rect');
    plotBg.setAttribute('x', String(mL));
    plotBg.setAttribute('y', String(mT));
    plotBg.setAttribute('width', String(pW));
    plotBg.setAttribute('height', String(pH));
    plotBg.setAttribute('fill', '#fff');
    plotBg.setAttribute('stroke', '#bbb');
    phaseChart.appendChild(plotBg);
    const xStep = niceStep(tMax - tMin, 6);
    const yStep = niceStep(yMax - yMin, 6);
    const xStart = Math.ceil(tMin / xStep) * xStep;
    const yStart = Math.ceil(yMin / yStep) * yStep;
    for (let xv = xStart; xv <= tMax + xStep * 0.01; xv += xStep) {
      const px = txp(xv);
      const grid = document.createElementNS(svgNs, 'line');
      grid.setAttribute('x1', String(px));
      grid.setAttribute('y1', String(mT));
      grid.setAttribute('x2', String(px));
      grid.setAttribute('y2', String(mT + pH));
      grid.setAttribute('stroke', '#e6e6e6');
      grid.setAttribute('stroke-width', '1');
      phaseChart.appendChild(grid);
      phaseChart.appendChild(svgText(px, mT + pH + 18, formatTick(xv, true), 'middle', 11));
    }
    for (let yv = yStart; yv <= yMax + yStep * 0.01; yv += yStep) {
      const py = typ(yv);
      const grid = document.createElementNS(svgNs, 'line');
      grid.setAttribute('x1', String(mL));
      grid.setAttribute('y1', String(py));
      grid.setAttribute('x2', String(mL + pW));
      grid.setAttribute('y2', String(py));
      grid.setAttribute('stroke', '#e6e6e6');
      grid.setAttribute('stroke-width', '1');
      phaseChart.appendChild(grid);
      phaseChart.appendChild(svgText(mL - 8, py + 4, formatTick(yv, false), 'end', 11));
    }
    const xAxis = document.createElementNS(svgNs, 'line');
    xAxis.setAttribute('x1', String(mL));
    xAxis.setAttribute('y1', String(mT + pH));
    xAxis.setAttribute('x2', String(mL + pW));
    xAxis.setAttribute('y2', String(mT + pH));
    xAxis.setAttribute('stroke', '#333');
    xAxis.setAttribute('stroke-width', '1.2');
    phaseChart.appendChild(xAxis);
    const yAxis = document.createElementNS(svgNs, 'line');
    yAxis.setAttribute('x1', String(mL));
    yAxis.setAttribute('y1', String(mT));
    yAxis.setAttribute('x2', String(mL));
    yAxis.setAttribute('y2', String(mT + pH));
    yAxis.setAttribute('stroke', '#333');
    yAxis.setAttribute('stroke-width', '1.2');
    phaseChart.appendChild(yAxis);
    phaseChart.appendChild(svgText(mL + pW / 2, H - 12, 't, с', 'middle', 13));
    phaseChart.appendChild(svgText(18, mT + pH / 2, 'Δφ, °', 'middle', 13, 'rotate(-90 18 ' + (mT + pH / 2) + ')'));
    phaseChart.appendChild(svgText(mL, 28, label, null, 13));
    let ptsStr = '';
    for (let i = 0; i < p.phaseT.length; i++) {
      ptsStr += txp(p.phaseT[i]) + ',' + typ(p.phaseDeg[i]) + ' ';
    }
    const poly = document.createElementNS(svgNs, 'polyline');
    poly.setAttribute('points', ptsStr.trim());
    poly.setAttribute('fill', 'none');
    poly.setAttribute('stroke', '#1565c0');
    poly.setAttribute('stroke-width', '1.8');
    phaseChart.appendChild(poly);
  }
  function show(p) {
    const x = esc(p.x);
    const y = esc(p.y);
    const v = esc(p.d);
    if (data.showOscMetrics) {
      let html =
        koef1Label + ' = ' + x + '<br>' +
        koef2Label + ' = ' + y;
      const deltaBlock = [];
      const freqBlock = [];
      if (p.delta != null) {
        deltaBlock.push(esc('delta') + ' = ' + p.delta);
      }
      if (p.delta != null && p.freq != null) {
        deltaBlock.push(esc('delta * frequency') + ' = ' + (p.delta * p.freq));
      }
      if (p.freq != null) {
        freqBlock.push(esc('frequency') + ' = ' + p.freq);
      }
      if (p.freq != null) {
        freqBlock.push(esc('frequency * 2π') + ' = ' + (2 * Math.PI * p.freq));
      }
      const blocks = data.mapMetric === 'frequency'
        ? [freqBlock, deltaBlock]
        : [deltaBlock, freqBlock];
      blocks.forEach(function(block) {
        block.forEach(function(line) {
          html += '<br>' + line;
        });
      });
      info.innerHTML = html;
      drawPhaseChart(p);
      return;
    }
    if (data.labelsUseHtml) {
      let html =
        koef1Label + ' = ' + x + '<br>' +
        koef2Label + ' = ' + y + '<br>' +
        esc(valueLabel) + ' = ' + v;
      info.innerHTML = html;
      return;
    }
    info.textContent =
      koef1Label + ' = ' + p.x + '\n' +
      koef2Label + ' = ' + p.y + '\n' +
      valueLabel + ' = ' + p.d;
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
    html_final = (
        html_tmpl.replace("__PAGE_TITLE__", page_title_esc)
        .replace("__NODE__", node_esc)
        .replace("__HINT__", hint_esc)
        .replace("__PAYLOAD__", payload_json)
    )

    out.write_text(html_final, encoding="utf-8")
    return fname


def _html_aero_maps_section(title: str, empty_msg: str, map_imgs: list[tuple]) -> str:
    if not map_imgs:
        return f"<p>{html_lib.escape(empty_msg)}</p>"
    parts = []
    for node_id, img_name, ih in map_imgs:
        parts.append(f"<h3>Узел {html_lib.escape(str(node_id))}</h3>")
        if ih:
            parts.append(
                f'<p><a href="{html_lib.escape(ih)}" target="_blank" rel="noopener">'
                "Интерактивная карта (клик по точке)</a></p>"
            )
        parts.append(
            f'<img src="{html_lib.escape(img_name)}" '
            f'alt="{html_lib.escape(title)} node {html_lib.escape(str(node_id))}"><br><br>'
        )
    return "".join(parts)


def build_aero_coef_map(
    plots_dir: Path,
    node_id: str,
    bx: np.ndarray,
    fy: np.ndarray,
    values: np.ndarray,
    *,
    file_stem: str,
    plot_title: str,
    cbar_label: str,
    interactive_page_title: str,
    interactive_hint: str,
    interactive_value_label: str,
    distinguish_zero: bool,
    interactive_koef1_label: str = "koef1 (forward)",
    interactive_koef2_label: str = "koef2 (back)",
    interactive_labels_use_html: bool = False,
    interactive_map_metric: str | None = None,
    interactive_paired_freq: np.ndarray | None = None,
    interactive_paired_delta: np.ndarray | None = None,
    interactive_phase_lookup: dict[tuple[float, float], dict] | None = None,
) -> tuple[str, str | None]:
    """PNG + интерактивная HTML-карта по сетке (koef1, koef2) → value."""
    img_name = f"{file_stem}_node{node_id}.png"

    mask_zero_maxs, mask_one_max, mask_marker = _aero_map_sentinel_masks(values)
    bx_col, fy_col, v_col = bx[~mask_marker], fy[~mask_marker], values[~mask_marker]
    bx_red, fy_red = bx[mask_zero_maxs], fy[mask_zero_maxs]
    bx_yel, fy_yel = bx[mask_one_max], fy[mask_one_max]

    plt.figure(figsize=(8, 6))
    sc = None
    if bx_col.size > 0:
        vmin, vmax = float(np.min(v_col)), float(np.max(v_col))
        if not np.isfinite(vmin) or not np.isfinite(vmax):
            vmin, vmax = 0.0, 1.0
        if abs(vmax - vmin) < 1e-30:
            vmax = vmin + 1e-30
        norm = Normalize(vmin=vmin, vmax=vmax)
        if distinguish_zero:
            mask_v0 = np.isclose(v_col, 0.0, rtol=0.0, atol=1e-12)
            if np.any(mask_v0):
                sc = plt.scatter(
                    bx_col[mask_v0],
                    fy_col[mask_v0],
                    c=v_col[mask_v0],
                    cmap="viridis",
                    norm=norm,
                    marker="o",
                    s=80,
                    edgecolors="k",
                    linewidths=0.25,
                )
            if np.any(~mask_v0):
                sc_sq = plt.scatter(
                    bx_col[~mask_v0],
                    fy_col[~mask_v0],
                    c=v_col[~mask_v0],
                    cmap="viridis",
                    norm=norm,
                    marker="s",
                    s=80,
                    edgecolors="k",
                    linewidths=0.25,
                )
                sc = sc_sq
        else:
            sc = plt.scatter(
                bx_col,
                fy_col,
                c=v_col,
                cmap="viridis",
                norm=norm,
                marker="s",
                s=80,
                edgecolors="k",
                linewidths=0.25,
            )
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
    plt.title(plot_title)
    plt.grid(True, alpha=0.25)
    plt.gca().set_aspect("equal", adjustable="box")
    if sc is not None:
        cbar = plt.colorbar(sc)
        cbar.set_label(cbar_label)
    safe_tight_layout()

    plt.savefig(plots_dir / img_name, dpi=200)
    plt.close()

    ih = _write_aero_map_interactive_html(
        plots_dir,
        node_id,
        bx,
        fy,
        values,
        file_stem=file_stem,
        page_title=interactive_page_title,
        hint_html=interactive_hint,
        value_label=interactive_value_label,
        distinguish_zero=distinguish_zero,
        koef1_label=interactive_koef1_label,
        koef2_label=interactive_koef2_label,
        labels_use_html=interactive_labels_use_html,
        map_metric=interactive_map_metric,
        paired_freq=interactive_paired_freq,
        paired_delta=interactive_paired_delta,
        phase_lookup=interactive_phase_lookup,
    )
    return img_name, ih


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
    phase_img_name = "interblade_phase.png"
    amp_img_name = "amplitudes.png"
    energy_img_name = "sum_energy.png"
    dsum_energy_img_name = "dsum_energy.png"
    node_energy_img_name = "node_energy.png"
    node_d_energy_img_name = "node_d_energy.png"
    analytical_dadt_img_name = "analytical_dadt.png"
    d_amp_img_name = "d_amplitudes.png"
    dec_img_name = "decrement_deltas.png"
    decr_map_imgs = []
    freq_map_imgs = []
    energy_per_node_has_data = False
    d_energy_per_node_has_data = False
    analytical_dadt_has_data = False
    d_amp_has_data = False
    phase_has_data = False

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

    # ---------- Межлопаточная фаза Δφ(t), градусы (single run) ----------
    if (not PARALLEL_ONLY_DECREMENT_MAPS) and graph_files and movable_node_ids:
        config_name = os.environ.get("CONFIG", "conf")
        config_path = root_dir / "internal" / "config" / "confs" / f"{config_name}.json"
        cfg: dict | None = None
        if config_path.exists():
            try:
                with config_path.open("r", encoding="utf-8") as f:
                    cfg = json.load(f)
            except Exception:
                cfg = None
        trajectories: dict[int, tuple[np.ndarray, np.ndarray]] = {}
        for path in graph_files:
            m = re.match(r"^graph_points(\d+)$", path.stem)
            if not m:
                continue
            t, x = load_two_column_txt(path)
            if t.size == 0:
                continue
            trajectories[int(m.group(1))] = (t, x)
        if cfg is not None and trajectories:
            plt.figure(figsize=(10, 6))
            colors = plt.cm.tab10.colors
            for idx, node_id in enumerate(movable_node_ids):
                neighbor_id = movable_node_ids[(idx + 1) % len(movable_node_ids)]
                if node_id not in trajectories or neighbor_id not in trajectories:
                    continue
                x_eq_a = equilibrium_for_node_from_config(cfg, node_id)
                x_eq_b = equilibrium_for_node_from_config(cfg, neighbor_id)
                if x_eq_a is None or x_eq_b is None:
                    continue
                t_a, x_a = trajectories[node_id]
                t_b, x_b = trajectories[neighbor_id]
                n = min(t_a.size, t_b.size)
                t_ph, ph = interblade_phase_diff_deg_numpy(
                    t_a[:n], x_a[:n], x_b[:n], x_eq_a, x_eq_b
                )
                if t_ph.size == 0:
                    continue
                phase_has_data = True
                plt.plot(
                    t_ph,
                    ph,
                    label=f"φ({node_id})−φ({neighbor_id})",
                    color=colors[idx % len(colors)],
                )
            if phase_has_data:
                plt.xlabel("t")
                plt.ylabel("Δφ, °")
                plt.title("Межлопаточная разность фаз (развёрнутая)")
                plt.grid(True, alpha=0.3)
                plt.legend()
                safe_tight_layout()
                plt.savefig(plots_dir / phase_img_name, dpi=200)
            plt.close()

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

    # ---------- Карты декремента и частоты по аэрокоэффициентам (PARALLEL=true) ----------
    if PARALLEL_ONLY_DECREMENT_MAPS:
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
            node_id = path.stem.replace("decrementStore", "")
            freq_paired: np.ndarray | None = None
            freq_path = data_dir / f"freqStore{node_id}.txt"
            if freq_path.exists():
                fbx, ffy, freq = load_three_column_txt(freq_path)
                if fbx.size > 0:
                    freq_lookup = _build_aero_coef_lookup(fbx, ffy, freq)
                    freq_paired = _lookup_aero_coef_values(freq_lookup, bx, fy)
            phase_lookup = load_phase_diff_lookup(data_dir / f"phaseDiffStore{node_id}.json")
            img_name, ih = build_aero_coef_map(
                plots_dir,
                node_id,
                bx,
                fy,
                delta,
                file_stem="decrement_map",
                plot_title=f"Mean decrement map for node {node_id}",
                cbar_label="delta",
                interactive_page_title="Карта декремента",
                interactive_hint=(
                    "Клик по квадрату: метрики и Δφ(t) для пары этой лопатки с соседом по кольцу. "
                    "Другую лопатку — на её карте (node 1 → φ(1)−φ(2), …)."
                ),
                interactive_value_label="delta",
                distinguish_zero=True,
                interactive_koef1_label="k<sup>+</sup>",
                interactive_koef2_label="k<sup>−</sup>",
                interactive_labels_use_html=True,
                interactive_map_metric="delta",
                interactive_paired_freq=freq_paired,
                interactive_phase_lookup=phase_lookup or None,
            )
            decr_map_imgs.append((node_id, img_name, ih or ""))

        if decr_store_files and not decr_store_has_data:
            print(f"Файлы decrementStore*.txt найдены, но пустые: {data_dir}")
        elif not decr_store_files:
            print(f"Файлы decrementStore*.txt не найдены в {data_dir}")

        freq_store_files = []
        if movable_node_ids:
            for node_id in movable_node_ids:
                path = data_dir / f"freqStore{node_id}.txt"
                if path.exists():
                    freq_store_files.append(path)
        else:
            freq_store_files = sorted(data_dir.glob("freqStore*.txt"))
        freq_store_has_data = False

        for path in freq_store_files:
            bx, fy, freq = load_three_column_txt(path)
            if bx.size == 0:
                continue

            freq_store_has_data = True
            node_id = path.stem.replace("freqStore", "")
            delta_paired: np.ndarray | None = None
            decr_path = data_dir / f"decrementStore{node_id}.txt"
            if decr_path.exists():
                dbx, dfy, dval = load_three_column_txt(decr_path)
                if dbx.size > 0:
                    delta_lookup = _build_aero_coef_lookup(dbx, dfy, dval)
                    delta_paired = _lookup_aero_coef_values(delta_lookup, bx, fy)
            phase_lookup = load_phase_diff_lookup(data_dir / f"phaseDiffStore{node_id}.json")
            img_name, ih = build_aero_coef_map(
                plots_dir,
                node_id,
                bx,
                fy,
                freq,
                file_stem="frequency_map",
                plot_title=f"Mean frequency map for node {node_id}",
                cbar_label="frequency (1/s)",
                interactive_page_title="Карта частоты",
                interactive_hint=(
                    "Клик по квадрату: метрики и Δφ(t) для пары этой лопатки с соседом по кольцу."
                ),
                interactive_value_label="frequency",
                distinguish_zero=False,
                interactive_koef1_label="k<sup>+</sup>",
                interactive_koef2_label="k<sup>−</sup>",
                interactive_labels_use_html=True,
                interactive_map_metric="frequency",
                interactive_paired_delta=delta_paired,
                interactive_phase_lookup=phase_lookup or None,
            )
            freq_map_imgs.append((node_id, img_name, ih or ""))

        if freq_store_files and not freq_store_has_data:
            print(f"Файлы freqStore*.txt найдены, но пустые: {data_dir}")
        elif not freq_store_files:
            print(f"Файлы freqStore*.txt не найдены в {data_dir}")

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
    <h2>Межлопаточная фаза Δφ(t), °</h2>
    {"<p>Нет graph_points* или подвижных узлов в конфиге.</p>" if not phase_has_data else f'<img src="{phase_img_name}" alt="Inter-blade phase">'}
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

  {f'''
  <div class="block">
    <h2>Карты декремента по аэрокоэффициентам (decrementStore*.txt)</h2>
    {_html_aero_maps_section(
        "Decrement map",
        "Файлы не найдены или пусты.",
        decr_map_imgs,
    )}
  </div>

  <div class="block">
    <h2>Карты частоты по аэрокоэффициентам (freqStore*.txt)</h2>
    {_html_aero_maps_section(
        "Frequency map",
        "Файлы не найдены или пусты.",
        freq_map_imgs,
    )}
  </div>
  ''' if PARALLEL_ONLY_DECREMENT_MAPS else ''}

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

