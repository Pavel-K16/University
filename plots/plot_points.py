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

import matplotlib.pyplot as plt
import numpy as np


def load_two_column_txt(path: Path):
    """
    Ожидается файл с двумя столбцами: t, value.
    Возвращает (t, y) как numpy-массивы.
    """
    data = np.loadtxt(path)
    if data.ndim == 1:
        data = data[None, :]
    return data[:, 0], data[:, 1]


def generate_plots_and_html():
    # Текущий файл: <root>/plots/plot_points.py
    plots_dir = Path(__file__).resolve().parent
    root_dir = plots_dir.parent

    data_dir = root_dir / "wolfram" / "paramsAndPoints"
    if not data_dir.exists():
        raise FileNotFoundError(f"Директория с данными не найдена: {data_dir}")

    plots_dir.mkdir(parents=True, exist_ok=True)

    # ---------- Траектории graph_points*.txt ----------
    graph_files = sorted(data_dir.glob("graph_points*.txt"))

    traj_img_name = "trajectories.png"
    energy_img_name = "sum_energy.png"

    if graph_files:
        plt.figure(figsize=(10, 6))
        colors = plt.cm.tab10.colors

        for idx, path in enumerate(graph_files):
            t, x = load_two_column_txt(path)
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
        print(f"Траектории сохранены в {out_traj}")
    else:
        print(f"Файлы graph_points*.txt не найдены в {data_dir}")

    # ---------- Суммарная энергия sumEnergyPoints.txt ----------
    energy_path = data_dir / "sumEnergyPoints.txt"
    energy_exists = energy_path.exists()

    if energy_exists:
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
        print(f"График энергии сохранён в {out_energy}")
    else:
        print(f"Файл с энергией не найден: {energy_path}")

    # ---------- Генерация HTML ----------
    html_path = plots_dir / "index.html"

    html = f"""<!DOCTYPE html>
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
  </style>
</head>
<body>
  <h1>Графики из wolfram/paramsAndPoints</h1>

  <div class="block">
    <h2>Траектории узлов (graph_points*.txt)</h2>
    {"<p>Файлы не найдены.</p>" if not graph_files else f'<img src="{traj_img_name}" alt="Trajectories">'}
  </div>

  <div class="block">
    <h2>Суммарная энергия (sumEnergyPoints.txt)</h2>
    {"<p>Файл не найден.</p>" if not energy_exists else f'<img src="{energy_img_name}" alt="Total energy">'}
  </div>
</body>
</html>
"""

    html_path.write_text(html, encoding="utf-8")
    print(f"HTML сохранён в {html_path}")


if __name__ == "__main__":
    generate_plots_and_html()

