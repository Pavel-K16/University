#!/usr/bin/env python3
"""
Экспорт амплитуд, суммарной энергии, декремента и межлопаточной фазы
в стиле Wolfram Mathematica → doc/images/.

Данные: wolfram/paramsAndPoints/
  amplitude_points<N>.txt
  sumEnergyPoints.txt
  decrement_points<N>.txt
  interblade_phase_points<N>.txt

Пример:
  scripts/export_metrics_images.sh 4blades
  scripts/export_metrics_images.sh 4blades --export-only
"""

from __future__ import annotations

import argparse
import os
from dataclasses import dataclass
from pathlib import Path

import matplotlib.pyplot as plt
import numpy as np

from export_trajectory import (
    BLADE_COLORS,
    CURVE_COLOR,
    CURVE_WIDTH,
    DEFAULT_DATA_DIR,
    DEFAULT_OUT_DIR,
    DPI,
    FIGSIZE,
    MULTI_CURVE_WIDTH,
    MULTI_FIGSIZE,
    _auto_y_limits,
    _suggest_integer_y_ticks,
    apply_mathematica_style,
    load_movable_node_ids,
    load_t_end_from_config,
    load_trajectory,
    save_wolfram_figure,
)

MARKER_SIZE = 45


@dataclass
class Series:
    label: str
    t: np.ndarray
    y: np.ndarray
    markers: bool = False


def _resolve_t_max(t_max: float | None, series: list[Series]) -> float:
    if t_max is not None:
        return t_max
    cfg_t = load_t_end_from_config()
    if cfg_t is not None:
        return cfg_t
    return max(float(np.max(s.t)) for s in series if s.t.size)


def _build_figure(
    series_list: list[Series],
    *,
    y_label: str,
    t_max: float | None = None,
    y_lim: tuple[float, float] | None = None,
    single_red: bool = False,
    figsize: tuple[float, float] = MULTI_FIGSIZE,
) -> tuple[plt.Figure, plt.Legend | None, list[plt.Annotation]]:
    if not series_list:
        raise ValueError("нет данных для графика")

    fig, ax = plt.subplots(figsize=figsize, dpi=DPI)
    handles: list[plt.Line2D] = []
    all_y: list[np.ndarray] = []

    for idx, s in enumerate(series_list):
        if s.t.size == 0:
            continue
        color = CURVE_COLOR if single_red else BLADE_COLORS[idx % len(BLADE_COLORS)]
        width = CURVE_WIDTH if single_red else MULTI_CURVE_WIDTH
        (line,) = ax.plot(
            s.t,
            s.y,
            color=color,
            linewidth=width,
            solid_capstyle="round",
            label=s.label,
            zorder=2 + idx,
        )
        if s.markers:
            ax.scatter(
                s.t,
                s.y,
                s=MARKER_SIZE,
                color=color,
                edgecolors=color,
                linewidths=0.8,
                zorder=4 + idx,
                clip_on=False,
            )
        handles.append(line)
        all_y.append(s.y)

    t_end = _resolve_t_max(t_max, series_list)
    if y_lim is None:
        y_lim = _auto_y_limits(all_y)
    y_ticks = _suggest_integer_y_ticks(y_lim[0], y_lim[1])
    axis_labels = apply_mathematica_style(
        ax,
        t_max=t_end,
        y_lim=y_lim,
        y_ticks=y_ticks,
        y_label=y_label,
    )

    legend = None
    if handles and not single_red:
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
    right = 0.98 if single_red else 0.64
    fig.subplots_adjust(left=0.10, right=right, top=0.90, bottom=0.14)
    return fig, legend, axis_labels


def _load_blade_series(
    data_dir: Path,
    template: str,
    node_ids: list[int],
    *,
    markers: bool = False,
) -> list[Series]:
    out: list[Series] = []
    for node_id in node_ids:
        path = data_dir / template.format(node_id)
        if not path.exists():
            print(f"Пропуск: нет файла {path}")
            continue
        t, y = load_trajectory(path)
        out.append(Series(f"лопатка {node_id}", t, y, markers=markers))
    return out


def export_amplitudes(
    data_dir: Path,
    out_dir: Path,
    node_ids: list[int],
    *,
    t_max: float | None,
) -> None:
    series = _load_blade_series(data_dir, "amplitude_points{}.txt", node_ids, markers=True)
    if not series:
        print("Амплитуды: нет данных")
        return
    fig, legend, axis_labels = _build_figure(series, y_label="A", t_max=t_max)
    save_wolfram_figure(fig, out_dir / "amplitudes", legend=legend, axis_labels=axis_labels)


def export_sum_energy(
    data_dir: Path,
    out_dir: Path,
    *,
    t_max: float | None,
) -> None:
    path = data_dir / "sumEnergyPoints.txt"
    if not path.exists():
        print(f"Суммарная энергия: нет файла {path}")
        return
    t, e = load_trajectory(path)
    fig, _, axis_labels = _build_figure(
        [Series("E", t, e)],
        y_label="E",
        t_max=t_max,
        single_red=True,
        figsize=FIGSIZE,
    )
    save_wolfram_figure(fig, out_dir / "sum_energy", axis_labels=axis_labels)


def export_decrements(
    data_dir: Path,
    out_dir: Path,
    node_ids: list[int],
    *,
    t_max: float | None,
) -> None:
    series = _load_blade_series(data_dir, "decrement_points{}.txt", node_ids, markers=True)
    if not series:
        print("Декремент: нет данных")
        return
    fig, legend, axis_labels = _build_figure(series, y_label="δ", t_max=t_max)
    save_wolfram_figure(fig, out_dir / "decrement_deltas", legend=legend, axis_labels=axis_labels)


def export_interblade_phase(
    data_dir: Path,
    out_dir: Path,
    node_ids: list[int],
    *,
    t_max: float | None,
) -> None:
    series: list[Series] = []
    for idx, node_id in enumerate(node_ids):
        neighbor_id = node_ids[(idx + 1) % len(node_ids)]
        path = data_dir / f"interblade_phase_points{node_id}.txt"
        if not path.exists():
            print(f"Пропуск: нет файла {path}")
            continue
        t, ph = load_trajectory(path)
        series.append(Series(f"φ({neighbor_id})−φ({node_id})", t, ph))
    if not series:
        print("Межлопаточная фаза: нет данных")
        return
    fig, legend, axis_labels = _build_figure(series, y_label="Δφ", t_max=t_max)
    save_wolfram_figure(fig, out_dir / "interblade_phase", legend=legend, axis_labels=axis_labels)


def export_all(
    data_dir: Path,
    out_dir: Path,
    *,
    config_name: str | None,
    t_max: float | None,
) -> None:
    if config_name:
        os.environ["CONFIG"] = config_name
    node_ids = load_movable_node_ids(config_name)
    if not node_ids:
        print("Предупреждение: не найден конфиг, узлы не определены")
    export_amplitudes(data_dir, out_dir, node_ids, t_max=t_max)
    export_sum_energy(data_dir, out_dir, t_max=t_max)
    export_decrements(data_dir, out_dir, node_ids, t_max=t_max)
    export_interblade_phase(data_dir, out_dir, node_ids, t_max=t_max)


def main() -> None:
    parser = argparse.ArgumentParser(description="Экспорт метрик в стиле Mathematica")
    parser.add_argument("--data-dir", type=Path, default=DEFAULT_DATA_DIR)
    parser.add_argument("--output-dir", type=Path, default=DEFAULT_OUT_DIR)
    parser.add_argument("--config", type=str, default=None)
    parser.add_argument("--t-max", type=float, default=None)
    parser.add_argument(
        "--only",
        choices=["amplitudes", "energy", "decrement", "phase", "all"],
        default="all",
    )
    args = parser.parse_args()
    data_dir = args.data_dir.resolve()
    out_dir = args.output_dir.resolve()
    if args.config:
        os.environ["CONFIG"] = args.config
    node_ids = load_movable_node_ids(args.config)

    if args.only == "all":
        export_all(data_dir, out_dir, config_name=args.config, t_max=args.t_max)
        return
    if args.only == "amplitudes":
        export_amplitudes(data_dir, out_dir, node_ids, t_max=args.t_max)
    elif args.only == "energy":
        export_sum_energy(data_dir, out_dir, t_max=args.t_max)
    elif args.only == "decrement":
        export_decrements(data_dir, out_dir, node_ids, t_max=args.t_max)
    elif args.only == "phase":
        export_interblade_phase(data_dir, out_dir, node_ids, t_max=args.t_max)


if __name__ == "__main__":
    main()
