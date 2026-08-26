#!/usr/bin/env python3
"""
Экспорт траекторий graph_points*.txt в стиле Wolfram Mathematica (красная кривая, оси t/x).

Данные: ../wolfram/paramsAndPoints/graph_points<N>.txt  (две колонки: t, x)

По умолчанию экспорт в doc/images.

Примеры:
  scripts/export_trajectory_images.sh
  scripts/export_trajectory_images.sh 1blade
  python3 export_trajectory.py --all
  python3 export_trajectory.py --node 0
  python3 export_trajectory.py --combined --config 4blades
"""

from __future__ import annotations

import argparse
import json
import os
import re
from pathlib import Path

import matplotlib.pyplot as plt
from matplotlib.ticker import MaxNLocator
import numpy as np

SCRIPT_DIR = Path(__file__).resolve().parent
ROOT_DIR = SCRIPT_DIR.parent
DEFAULT_DATA_DIR = ROOT_DIR / "wolfram" / "paramsAndPoints"
DEFAULT_OUT_DIR = ROOT_DIR / "doc" / "images"
CONFS_DIR = ROOT_DIR / "internal" / "config" / "confs"

# Стиль как на эталонном графике: красная жирная линия, белый фон, без сетки.
CURVE_COLOR = "#E41A1C"
CURVE_WIDTH = 2.8
MULTI_CURVE_WIDTH = 2.4
FIGSIZE = (7.2, 4.8)
MULTI_FIGSIZE = (9.2, 5.2)
DPI = 200
BLADE_COLORS = list(plt.cm.tab10.colors)


def load_trajectory(path: Path) -> tuple[np.ndarray, np.ndarray]:
    if not path.exists():
        raise FileNotFoundError(path)
    data = np.loadtxt(path)
    if data.ndim == 1:
        if data.size < 2:
            return np.array([]), np.array([])
        return np.array([data[0]]), np.array([data[1]])
    return data[:, 0], data[:, 1]


def config_name_from_env() -> str:
    return os.environ.get("CONFIG", "conf")


def load_movable_node_ids(config_name: str | None = None) -> list[int]:
    name = config_name or config_name_from_env()
    path = CONFS_DIR / f"{name}.json"
    if not path.exists():
        return []
    cfg = json.loads(path.read_text(encoding="utf-8"))
    return sorted(
        int(n["id"]) for n in cfg.get("nodes", []) if not n.get("fixed", False)
    )


def load_t_end_from_config(config_name: str | None = None) -> float | None:
    name = config_name or config_name_from_env()
    path = CONFS_DIR / f"{name}.json"
    if not path.exists():
        return None
    cfg = json.loads(path.read_text(encoding="utf-8"))
    t_end = (cfg.get("times") or {}).get("t")
    return float(t_end) if t_end is not None else None


def discover_graph_point_files(data_dir: Path) -> list[Path]:
    out: list[tuple[int, Path]] = []
    for p in sorted(data_dir.glob("graph_points*.txt")):
        m = re.match(r"^graph_points(\d+)$", p.stem)
        if m:
            out.append((int(m.group(1)), p))
    return [path for _, path in sorted(out)]


def apply_mathematica_style(
    ax: plt.Axes,
    *,
    t_max: float | None = None,
    y_lim: tuple[float, float] | None = None,
    y_ticks: np.ndarray | None = None,
    y_label: str = "x",
    x_label: str = "t",
    label_fontsize: float = 13,
    tick_fontsize: float = 12,
) -> list[plt.Annotation]:
    """Оформление осей в духе Mathematica: t вправо, подпись y вверх, без сетки."""
    ax.grid(False)
    ax.set_facecolor("white")

    xmin, xmax = ax.get_xlim()
    ymin, ymax = ax.get_ylim()

    if t_max is not None:
        xmax = float(t_max)
        ax.set_xlim(0.0, xmax)
    else:
        ax.set_xlim(max(0.0, xmin), xmax)

    if y_lim is not None:
        ax.set_ylim(y_lim[0], y_lim[1])
        ymin, ymax = y_lim

    # Оси через начало координат (крест), без верхней и правой «рамки».
    ax.spines["top"].set_visible(False)
    ax.spines["right"].set_visible(False)
    ax.spines["left"].set_position(("data", 0.0))
    ax.spines["bottom"].set_position(("data", 0.0))

    # Деления как на эталоне: 10, 20, … по t; -2, -1, 1, 2 по x (если влезают).
    if xmax > 0:
        step = 10.0
        xticks = np.arange(step, xmax + 0.1, step)
        if xticks.size == 0 or xticks[-1] < xmax * 0.5:
            xticks = np.linspace(0, xmax, min(6, int(xmax) + 1))
        ax.set_xticks(xticks)

    if y_ticks is not None and y_ticks.size:
        ax.set_yticks(y_ticks)
    else:
        span = ymax - ymin
        if span <= 5.0:
            yticks_candidate = np.array([-2.0, -1.0, 1.0, 2.0])
            yticks = yticks_candidate[(yticks_candidate >= ymin) & (yticks_candidate <= ymax)]
            if yticks.size:
                ax.set_yticks(yticks)
        else:
            ax.yaxis.set_major_locator(MaxNLocator(nbins=6, min_n_ticks=4))

    ax.tick_params(axis="both", direction="out", length=5, width=0.9, colors="black", labelsize=tick_fontsize)

    # Подписи у концов осей: t — справа от горизонтальной оси, x — над вертикальной.
    ax.set_xlabel("")
    ax.set_ylabel("")
    xmax = ax.get_xlim()[1]
    t_label = ax.annotate(
        x_label,
        xy=(xmax, 0.0),
        xycoords="data",
        xytext=(10, 0),
        textcoords="offset points",
        ha="left",
        va="center",
        fontsize=label_fontsize,
        fontstyle="italic",
        clip_on=False,
    )
    y_axis_label = ax.annotate(
        y_label,
        xy=(0.0, ymax),
        xycoords="data",
        xytext=(0, 10),
        textcoords="offset points",
        ha="center",
        va="bottom",
        fontsize=label_fontsize,
        fontstyle="italic",
        clip_on=False,
    )
    return [t_label, y_axis_label]


def save_wolfram_figure(
    fig: plt.Figure,
    output_path: Path,
    *,
    legend: plt.Legend | None = None,
    axis_labels: list[plt.Annotation] | None = None,
    formats: list[str] | None = None,
) -> None:
    extra: list = []
    if axis_labels:
        extra.extend(axis_labels)
    if legend is not None:
        extra.append(legend)
    save_kwargs: dict = {
        "dpi": DPI,
        "facecolor": "white",
        "edgecolor": "none",
        "bbox_inches": "tight",
        "pad_inches": 0.10,
    }
    if extra:
        save_kwargs["bbox_extra_artists"] = extra
    output_path.parent.mkdir(parents=True, exist_ok=True)
    stem = output_path.with_suffix("")
    for fmt in formats or ["png"]:
        out = stem.with_suffix(f".{fmt}")
        fig.savefig(out, **save_kwargs)
        print(f"Сохранено: {out}")
    plt.close(fig)


def plot_trajectory_mathematica(
    t: np.ndarray,
    x: np.ndarray,
    *,
    t_max: float | None = None,
    y_lim: tuple[float, float] | None = None,
    title: str | None = None,
) -> plt.Figure:
    if t.size == 0:
        raise ValueError("пустая траектория")

    fig, ax = plt.subplots(figsize=FIGSIZE, dpi=DPI)
    ax.plot(t, x, color=CURVE_COLOR, linewidth=CURVE_WIDTH, solid_capstyle="round")

    if t_max is None:
        t_max = float(np.max(t))

    if y_lim is None:
        pad = 0.1 * max(1e-9, float(np.max(np.abs(x))))
        y_lim = (float(np.min(x)) - pad, float(np.max(x)) + pad)

    apply_mathematica_style(ax, t_max=t_max, y_lim=y_lim)

    if title:
        ax.set_title(title, fontsize=11, pad=8)

    fig.patch.set_facecolor("white")
    fig.subplots_adjust(left=0.12, right=0.98, top=0.96, bottom=0.14)
    return fig


def _suggest_integer_y_ticks(ymin: float, ymax: float) -> np.ndarray | None:
    """Целые деления по x для нескольких лопаток (как в Wolfram ListLinePlot)."""
    span = ymax - ymin
    if span > 8.0:
        return None
    lo = int(np.floor(ymin))
    hi = int(np.ceil(ymax))
    ticks = np.arange(lo, hi + 1, dtype=float)
    ticks = ticks[(ticks >= ymin - 1e-9) & (ticks <= ymax + 1e-9)]
    return ticks if ticks.size >= 2 else None


def _auto_y_limits(series: list[np.ndarray]) -> tuple[float, float]:
    if not series:
        return (-1.0, 1.0)
    stacked = np.concatenate([s for s in series if s.size])
    ymin = float(np.min(stacked))
    ymax = float(np.max(stacked))
    pad = 0.08 * max(1e-9, ymax - ymin, abs(ymax), abs(ymin))
    y0 = min(0.0, ymin - pad)
    return y0, ymax + pad


def plot_all_trajectories_mathematica(
    blades: list[tuple[int, np.ndarray, np.ndarray]],
    *,
    t_max: float | None = None,
    y_lim: tuple[float, float] | None = None,
) -> tuple[plt.Figure, plt.Legend | None, list[plt.Annotation]]:
    if not blades:
        raise ValueError("нет траекторий для экспорта")

    fig, ax = plt.subplots(figsize=MULTI_FIGSIZE, dpi=DPI)
    handles: list[plt.Line2D] = []
    all_x: list[np.ndarray] = []

    for idx, (node_id, t, x) in enumerate(blades):
        if t.size == 0:
            continue
        color = BLADE_COLORS[idx % len(BLADE_COLORS)]
        (line,) = ax.plot(
            t,
            x,
            color=color,
            linewidth=MULTI_CURVE_WIDTH,
            solid_capstyle="round",
            label=f"лопатка {node_id}",
            zorder=2 + idx,
        )
        handles.append(line)
        all_x.append(x)

    if t_max is None:
        t_max = max(float(np.max(t)) for _, t, _ in blades if t.size)

    if y_lim is None:
        y_lim = _auto_y_limits(all_x)

    y_ticks = _suggest_integer_y_ticks(y_lim[0], y_lim[1])
    axis_labels = apply_mathematica_style(ax, t_max=t_max, y_lim=y_lim, y_ticks=y_ticks)

    legend = None
    if handles:
        legend = ax.legend(
            handles,
            [h.get_label() for h in handles],
            loc="upper left",
            bbox_to_anchor=(1.05, 1.0),
            frameon=True,
            fancybox=True,
            fontsize=11,
            borderaxespad=0.0,
            handlelength=2.4,
        )

    fig.patch.set_facecolor("white")
    fig.subplots_adjust(left=0.10, right=0.64, top=0.90, bottom=0.14)
    return fig, legend, axis_labels


def export_combined(
    data_dir: Path,
    output_path: Path,
    *,
    node_ids: list[int] | None,
    t_max: float | None,
    y_lim: tuple[float, float] | None,
    formats: list[str],
) -> None:
    ids = node_ids if node_ids else load_movable_node_ids()
    if not ids:
        ids = []
        for path in discover_graph_point_files(data_dir):
            m = re.search(r"(\d+)$", path.stem)
            if m:
                ids.append(int(m.group(1)))

    blades: list[tuple[int, np.ndarray, np.ndarray]] = []
    for node_id in ids:
        path = data_dir / f"graph_points{node_id}.txt"
        if not path.exists():
            print(f"Пропуск: нет файла {path}")
            continue
        t, x = load_trajectory(path)
        blades.append((node_id, t, x))

    if not blades:
        raise FileNotFoundError(f"нет graph_points*.txt для узлов {ids} в {data_dir}")

    if t_max is None:
        t_max = load_t_end_from_config()

    fig, legend, axis_labels = plot_all_trajectories_mathematica(
        blades, t_max=t_max, y_lim=y_lim
    )
    save_wolfram_figure(
        fig,
        output_path,
        legend=legend,
        axis_labels=axis_labels,
        formats=formats,
    )


def export_one(
    graph_path: Path,
    output_path: Path,
    *,
    t_max: float | None,
    y_lim: tuple[float, float] | None,
    formats: list[str],
) -> None:
    t, x = load_trajectory(graph_path)
    fig = plot_trajectory_mathematica(t, x, t_max=t_max, y_lim=y_lim)
    output_path.parent.mkdir(parents=True, exist_ok=True)

    stem = output_path.with_suffix("")
    for fmt in formats:
        out = stem.with_suffix(f".{fmt}")
        fig.savefig(out, dpi=DPI, facecolor="white", edgecolor="none", bbox_inches="tight")
        print(f"Сохранено: {out}")

    plt.close(fig)


def main() -> None:
    parser = argparse.ArgumentParser(description="Экспорт graph_points в стиле Mathematica")
    parser.add_argument(
        "--data-dir",
        type=Path,
        default=DEFAULT_DATA_DIR,
        help="Каталог с graph_points*.txt",
    )
    parser.add_argument("--node", type=int, default=None, help="Номер узла (один файл)")
    parser.add_argument("--all", action="store_true", help="Экспорт всех graph_points*.txt")
    parser.add_argument(
        "--combined",
        action="store_true",
        help="Все подвижные лопатки на одном графике (легенда: лопатка N)",
    )
    parser.add_argument(
        "--config",
        type=str,
        default=None,
        help="Имя конфига для выбора подвижных узлов (по умолчанию CONFIG или conf)",
    )
    parser.add_argument(
        "--output",
        type=Path,
        default=None,
        help="Выходной файл (для одного узла), без расширения или с .png",
    )
    parser.add_argument(
        "--output-dir",
        type=Path,
        default=DEFAULT_OUT_DIR,
        help="Каталог для --all",
    )
    parser.add_argument(
        "--t-max",
        type=float,
        default=None,
        help="Правый предел оси t (по умолчанию max(t) из данных)",
    )
    parser.add_argument(
        "--y-min",
        type=float,
        default=-9999.0,
        help="Нижний предел оси x; по умолчанию авто по данным",
    )
    parser.add_argument(
        "--y-max",
        type=float,
        default=9999.0,
        help="Верхний предел оси x; по умолчанию авто по данным",
    )
    parser.add_argument(
        "--fixed-y",
        action="store_true",
        help="Фиксированная шкала y ∈ [-2.2, 2.2] (эталон одной лопатки)",
    )
    parser.add_argument(
        "--format",
        action="append",
        default=["png"],
        choices=["png", "pdf", "svg"],
        help="Формат(ы) экспорта (можно указать несколько раз)",
    )
    args = parser.parse_args()

    if args.fixed_y:
        y_lim: tuple[float, float] | None = (-2.2, 2.2)
    elif args.y_min <= -9998 or args.y_max >= 9998:
        y_lim = None
    else:
        y_lim = (args.y_min, args.y_max)

    data_dir = args.data_dir.resolve()

    if args.combined:
        if args.config:
            os.environ["CONFIG"] = args.config
        out = args.output if args.output is not None else args.output_dir / "trajectories"
        node_ids = load_movable_node_ids(args.config)
        export_combined(
            data_dir,
            out,
            node_ids=node_ids or None,
            t_max=args.t_max,
            y_lim=y_lim,
            formats=args.format,
        )
        return

    if args.all:
        files = discover_graph_point_files(data_dir)
        if not files:
            raise SystemExit(f"Нет graph_points*.txt в {data_dir}")
        for path in files:
            m = re.search(r"(\d+)$", path.stem)
            node_id = m.group(1) if m else path.stem
            out = args.output_dir / f"trajectory_node{node_id}"
            export_one(path, out, t_max=args.t_max, y_lim=y_lim, formats=args.format)
        return

    node = args.node if args.node is not None else 0
    graph_path = data_dir / f"graph_points{node}.txt"
    if args.output is not None:
        out_path = args.output
    else:
        out_path = args.output_dir / f"trajectory_node{node}"

    export_one(graph_path, out_path, t_max=args.t_max, y_lim=y_lim, formats=args.format)


if __name__ == "__main__":
    main()
