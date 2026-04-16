#!/usr/bin/env python3
"""
Генерация HTML-страницы с графиками траекторий и суммарной энергии.

Данные берутся из:
  ../wolfram/paramsAndPoints/graph_points*.txt   (t, x)
  ../wolfram/paramsAndPoints/sumEnergyPoints.txt (t, E)

Результаты сохраняются в:
  plots/trajectories.png
  plots/sum_energy.png
  plots/index.html
"""

from pathlib import Path
import os
import json
import html as html_lib

import matplotlib.pyplot as plt
import numpy as np

PARALLEL_ONLY_DECREMENT_MAPS = True


def load_two_column_txt(path: Path):
    """
    Ожидается файл с двумя столбцами: t, value.
    Возвращает (t, y) как numpy-массивы.
    """
    if path.stat().st_size == 0:
        return np.array([]), np.array([])
    data = np.loadtxt(path)
    if np.size(data) == 0:
        return np.array([]), np.array([])
    if data.ndim == 1:
        data = data[None, :]
    return data[:, 0], data[:, 1]


def load_three_column_txt(path: Path):
    """
    Ожидается файл с тремя столбцами: x, y, value.
    Возвращает (x, y, v) как numpy-массивы.
    """
    if path.stat().st_size == 0:
        return np.array([]), np.array([]), np.array([])
    data = np.loadtxt(path)
    if np.size(data) == 0:
        return np.array([]), np.array([]), np.array([])
    if data.ndim == 1:
        data = data[None, :]
    return data[:, 0], data[:, 1], data[:, 2]


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


def generate_plots_and_html():
    # Текущий файл: <root>/plots/plot_points.py
    plots_dir = Path(__file__).resolve().parent
    root_dir = plots_dir.parent

    data_dir = root_dir / "wolfram" / "paramsAndPoints"
    if not data_dir.exists():
        raise FileNotFoundError(f"Директория с данными не найдена: {data_dir}")

    plots_dir.mkdir(parents=True, exist_ok=True)

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
    dec_img_name = "decrement_deltas.png"
    decr_map_imgs = []

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
        plt.tight_layout()

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
        plt.tight_layout()

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
        plt.tight_layout()

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

        plt.figure(figsize=(8, 6))
        sc = plt.scatter(
            bx,
            fy,
            c=delta,
            cmap="viridis",
            s=80,
            edgecolors="k",
            linewidths=0.25,
        )
        plt.xlabel("koef1 (forward)")
        plt.ylabel("koef2 (back)")
        plt.title(f"Mean decrement map for node {node_id}")
        plt.grid(True, alpha=0.25)
        plt.gca().set_aspect("equal", adjustable="box")
        cbar = plt.colorbar(sc)
        cbar.set_label("delta")
        plt.tight_layout()

        out_map = plots_dir / img_name
        plt.savefig(out_map, dpi=200)
        plt.close()

        decr_map_imgs.append((node_id, img_name))

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
        plt.tight_layout()

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
        plt.tight_layout()

        out_dsum_energy = plots_dir / dsum_energy_img_name
        plt.savefig(out_dsum_energy, dpi=200)
        plt.close()
    elif not PARALLEL_ONLY_DECREMENT_MAPS:
        print(f"Файл с производной полной энергии не найден: {dsum_energy_path}")

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

    html_content = f"""<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="UTF-8">
  <title>Графики траекторий и энергии</title>
  <style>
    body {{
      font-family: sans-serif;
      margin: 20px;
      background: #fafafa;
    }}
    h1, h2 {{
      font-weight: 600;
    }}
    img {{
      max-width: 100%;
      height: auto;
      border: 1px solid #ccc;
      background: #fff;
      padding: 4px;
      box-shadow: 0 2px 4px rgba(0,0,0,0.1);
    }}
    .block {{
      margin-bottom: 32px;
    }}
    .formula-block math {{
      font-size: 1.55em;
    }}
    .formula-note {{
      margin-top: 8px;
      font-size: 1.08em;
      color: #202020;
    }}
  </style>
</head>
<body>
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
    {"<p>Файлы не найдены или пусты.</p>" if not decr_map_imgs else "".join(f'<h3>Узел {html_lib.escape(node_id)}</h3><img src="{html_lib.escape(img_name)}" alt="Decrement map node {html_lib.escape(node_id)}"><br><br>' for node_id, img_name in decr_map_imgs)}
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
</body>
</html>
"""

    html_path.write_text(html_content, encoding="utf-8")


if __name__ == "__main__":
    generate_plots_and_html()

