#!/usr/bin/env python3
"""
График погрешности численного решения err(τ) по таблице 1 (КПА, с. 10).

Данные: doc/images/table1.jpg — «Таблица 1. Погрешности численного решения
для разных τ» (D = 0.1, x₀ = 2, u₀ = 1, k = 2, m = 1).

Выход: doc/images/approximation_error.png

Пример:
  scripts/export_approximation_error.sh
"""

from __future__ import annotations

import argparse
from pathlib import Path

import matplotlib.pyplot as plt
import numpy as np

from export_trajectory import CURVE_COLOR, CURVE_WIDTH, DPI, FIGSIZE

SCRIPT_DIR = Path(__file__).resolve().parent
ROOT_DIR = SCRIPT_DIR.parent
DEFAULT_OUT_DIR = ROOT_DIR / "doc" / "images"

# Таблица 1, КПА с. 10 (значения err согласованы со столбцом err_{i-1}/err_i ≈ 16).
TABLE1_TAU = np.array([0.1, 0.05, 0.025, 0.0125, 0.00625])
TABLE1_ERR = np.array([2.57e-2, 1.61e-3, 1.01e-4, 6.33e-6, 3.96e-7])

X_MAX = 0.10
Y_MIN = 1e-7
Y_MAX = 1e-1
MARKER_COLOR = "black"
MARKER_SIZE = 70
AXIS_LABEL_FONTSIZE = 21
TICK_LABEL_FONTSIZE = 18
CURVE_WIDTH_ERR = 2.4


def apply_mathematica_log_style(
    ax: plt.Axes,
    *,
    x_max: float = X_MAX,
    y_min: float = Y_MIN,
    y_max: float = Y_MAX,
    x_label: str = "τ",
    y_label: str = "err",
    label_fontsize: float = AXIS_LABEL_FONTSIZE,
    tick_fontsize: float = TICK_LABEL_FONTSIZE,
) -> None:
    """Оформление log-графика в духе Mathematica: подписи у концов осей, без сетки."""
    ax.grid(False)
    ax.set_facecolor("white")
    ax.set_xlim(0.0, x_max)
    ax.set_ylim(y_min, y_max)
    ax.set_xticks(np.arange(0.02, x_max + 0.001, 0.02))
    ax.set_yticks([1e-2, 1e-3, 1e-4, 1e-5, 1e-6, 1e-7])

    ax.spines["top"].set_visible(False)
    ax.spines["right"].set_visible(False)
    ax.spines["left"].set_position(("data", 0.0))
    ax.spines["bottom"].set_position(("data", y_min))

    ax.tick_params(axis="both", direction="out", length=5, width=0.9, colors="black", labelsize=tick_fontsize)
    ax.set_xlabel("")
    ax.set_ylabel("")
    ax.annotate(
        x_label,
        xy=(x_max, y_min),
        xycoords="data",
        xytext=(10, 0),
        textcoords="offset points",
        ha="left",
        va="center",
        fontsize=label_fontsize,
        fontstyle="italic",
    )
    ax.annotate(
        y_label,
        xy=(0.0, y_max),
        xycoords="data",
        xytext=(0, 8),
        textcoords="offset points",
        ha="center",
        va="bottom",
        fontsize=label_fontsize,
        fontstyle="italic",
    )


def plot_approximation_error(
    *,
    tau: np.ndarray = TABLE1_TAU,
    err: np.ndarray = TABLE1_ERR,
    out_path: Path,
) -> None:
    fig, ax = plt.subplots(figsize=FIGSIZE, dpi=DPI)
    ax.set_yscale("log")
    ax.plot(
        tau,
        err,
        color=CURVE_COLOR,
        linewidth=CURVE_WIDTH_ERR,
        solid_capstyle="round",
        zorder=2,
    )
    ax.scatter(
        tau,
        err,
        s=MARKER_SIZE,
        c=MARKER_COLOR,
        edgecolors=MARKER_COLOR,
        linewidths=1.0,
        zorder=5,
        clip_on=False,
    )
    apply_mathematica_log_style(ax)

    fig.patch.set_facecolor("white")
    fig.subplots_adjust(left=0.12, right=0.96, top=0.96, bottom=0.14)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    fig.savefig(
        out_path,
        dpi=DPI,
        facecolor="white",
        edgecolor="none",
        bbox_inches="tight",
        pad_inches=0.06,
    )
    plt.close(fig)
    print(f"Сохранено: {out_path}")


def main() -> None:
    parser = argparse.ArgumentParser(description="График err(τ) по таблице 1")
    parser.add_argument("--output", type=Path, default=DEFAULT_OUT_DIR / "approximation_error.png")
    args = parser.parse_args()
    plot_approximation_error(out_path=args.output.resolve())


if __name__ == "__main__":
    main()
