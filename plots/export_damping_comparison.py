#!/usr/bin/env python3
"""
Сравнение численного (numSol) и аналитического (anSol) решений
для одной лопатки: m x'' + D x' + k x = 0.

Выход в doc/images:
  damping_D0.1.png, damping_D-0.1.png, damping_comparison.png

Пример:
  scripts/export_damping_comparison.sh
"""

from __future__ import annotations

import argparse
import json
import os
import shutil
import subprocess
from dataclasses import dataclass
from pathlib import Path

import matplotlib.pyplot as plt
from matplotlib.lines import Line2D
import numpy as np

from export_trajectory import DPI, apply_mathematica_style, load_trajectory

SCRIPT_DIR = Path(__file__).resolve().parent
ROOT_DIR = SCRIPT_DIR.parent
SCRIPTS_DIR = ROOT_DIR / "scripts"
DATA_DIR = ROOT_DIR / "wolfram" / "paramsAndPoints"
CONFS_DIR = ROOT_DIR / "internal" / "config" / "confs"
CACHE_DIR = ROOT_DIR / "wolfram" / "elementary_solution"
DEFAULT_OUT_DIR = ROOT_DIR / "doc" / "images"

NUM_COLOR = "#E41A1C"
NUM_WIDTH = 2.5
AN_COLOR = "black"
AN_WIDTH = 2.2
AN_DASH_ON = 10
AN_DASH_OFF = 5
AXIS_LABEL_FONTSIZE = 21
TICK_LABEL_FONTSIZE = 18
LEGEND_FONTSIZE = 16
CASE_TITLE_FONTSIZE = 20


@dataclass
class CaseData:
    config_name: str
    d_value: float
    t_end: float
    t_num: np.ndarray
    x_num: np.ndarray
    t_an: np.ndarray
    x_an: np.ndarray


def analytic_solution(
    t: np.ndarray,
    *,
    x0: float,
    v0: float,
    m: float,
    k: float,
    d: float,
) -> np.ndarray:
    """m x'' + d x' + k x = 0, x(0)=x0, x'(0)=v0."""
    wn = np.sqrt(k / m)
    zeta = d / (2.0 * wn)
    disc = 1.0 - zeta * zeta
    if disc <= 0:
        raise ValueError(f"неколебательный режим: zeta={zeta}")
    wd = wn * np.sqrt(disc)
    a = x0
    b = (v0 + zeta * wn * x0) / wd
    envelope = np.exp(-zeta * wn * t)
    return envelope * (a * np.cos(wd * t) + b * np.sin(wd * t))


def load_case_params(config_name: str) -> dict:
    path = CONFS_DIR / f"{config_name}.json"
    cfg = json.loads(path.read_text(encoding="utf-8"))
    blade = next(n for n in cfg["nodes"] if not n.get("fixed"))
    edge = next(e for e in cfg["edges"] if e["from"] == blade["id"])
    return {
        "t_end": float(cfg["times"]["t"]),
        "x0": float(blade["position"]),
        "v0": float(blade["velocity"]),
        "m": float(blade["mass"]),
        "k": float(edge["k"]),
        "d": float(edge["d"]),
    }


def run_nsolv(config_name: str) -> None:
    env = os.environ.copy()
    env["CONFIG"] = config_name
    env["PARALLEL"] = "false"
    subprocess.run(
        ["go", "run", "../cmd/nsolv.go"],
        cwd=SCRIPTS_DIR,
        env=env,
        check=True,
        capture_output=True,
    )


def cached_trajectory_path(config_name: str) -> Path:
    return CACHE_DIR / f"{config_name}_graph_points0.txt"


def load_case(config_name: str, *, run_solver: bool) -> CaseData:
    params = load_case_params(config_name)
    cache_path = cached_trajectory_path(config_name)
    if run_solver:
        print(f"Расчёт CONFIG={config_name} …")
        run_nsolv(config_name)
        CACHE_DIR.mkdir(parents=True, exist_ok=True)
        shutil.copy2(DATA_DIR / "graph_points0.txt", cache_path)
    elif not cache_path.exists():
        raise FileNotFoundError(
            f"Нет кэша {cache_path}; запустите без --no-run или сначала выполните расчёт."
        )
    t_num, x_num = load_trajectory(cache_path)
    n_dense = max(8000, len(t_num) * 4)
    t_an = np.linspace(0.0, params["t_end"], n_dense)
    x_an = analytic_solution(
        t_an,
        x0=params["x0"],
        v0=params["v0"],
        m=params["m"],
        k=params["k"],
        d=params["d"],
    )
    return CaseData(
        config_name=config_name,
        d_value=params["d"],
        t_end=params["t_end"],
        t_num=t_num,
        x_num=x_num,
        t_an=t_an,
        x_an=x_an,
    )


def _apply_analytical_dash(line: Line2D) -> None:
    line.set_linestyle("-")
    line.set_dashes([AN_DASH_ON, AN_DASH_OFF])
    line.set_dash_capstyle("butt")
    line.set_solid_capstyle("butt")


def _legend_handles() -> list[Line2D]:
    h_an = Line2D([0], [0], color=AN_COLOR, linewidth=AN_WIDTH)
    _apply_analytical_dash(h_an)
    h_num = Line2D([0], [0], color=NUM_COLOR, linewidth=NUM_WIDTH, linestyle="-")
    return [h_an, h_num]


def add_case_title(ax: plt.Axes, d_value: float) -> None:
    x_pos = 0.79 if d_value < 0 else 0.97
    ax.text(
        x_pos,
        0.94,
        rf"$D = {d_value:.1f}$",
        transform=ax.transAxes,
        ha="right",
        va="top",
        fontsize=CASE_TITLE_FONTSIZE,
        color="black",
        clip_on=False,
    )


def plot_panel(
    ax: plt.Axes,
    case: CaseData,
    *,
    y_lim: tuple[float, float] | None,
    show_legend: bool,
) -> None:
    (line_num,) = ax.plot(
        case.t_num,
        case.x_num,
        color=NUM_COLOR,
        linewidth=NUM_WIDTH,
        solid_capstyle="round",
        zorder=2,
    )
    (line_an,) = ax.plot(
        case.t_an,
        case.x_an,
        color=AN_COLOR,
        linewidth=AN_WIDTH,
        zorder=3,
        clip_on=True,
    )
    _apply_analytical_dash(line_an)
    apply_mathematica_style(
        ax,
        t_max=case.t_end,
        y_lim=y_lim,
        label_fontsize=AXIS_LABEL_FONTSIZE,
        tick_fontsize=TICK_LABEL_FONTSIZE,
    )
    add_case_title(ax, case.d_value)
    if show_legend:
        ax.legend(
            _legend_handles(),
            ["Аналитическое", "Численное"],
            loc="upper left",
            bbox_to_anchor=(1.06, 1.0),
            frameon=True,
            fancybox=True,
            fontsize=LEGEND_FONTSIZE,
            borderaxespad=0.0,
            handlelength=3.0,
            handleheight=1.4,
        )


def save_single(case: CaseData, path: Path, y_lim: tuple[float, float] | None) -> None:
    fig, ax = plt.subplots(figsize=(7.2, 4.2), dpi=DPI)
    plot_panel(ax, case, y_lim=y_lim, show_legend=True)
    fig.patch.set_facecolor("white")
    fig.subplots_adjust(left=0.10, right=0.68, top=0.96, bottom=0.12)
    path.parent.mkdir(parents=True, exist_ok=True)
    fig.savefig(
        path,
        dpi=DPI,
        facecolor="white",
        edgecolor="none",
        bbox_inches="tight",
        pad_inches=0.08,
    )
    plt.close(fig)
    print(f"Сохранено: {path}")


def save_combined(cases: list[tuple[CaseData, tuple[float, float] | None]], path: Path) -> None:
    fig, axes = plt.subplots(2, 1, figsize=(7.2, 8.4), dpi=DPI)
    for ax, (case, y_lim) in zip(axes, cases):
        plot_panel(ax, case, y_lim=y_lim, show_legend=True)
    fig.patch.set_facecolor("white")
    fig.subplots_adjust(left=0.10, right=0.68, top=0.98, bottom=0.08, hspace=0.38)
    path.parent.mkdir(parents=True, exist_ok=True)
    fig.savefig(
        path,
        dpi=DPI,
        facecolor="white",
        edgecolor="none",
        bbox_inches="tight",
        pad_inches=0.08,
    )
    plt.close(fig)
    print(f"Сохранено: {path}")


def main() -> None:
    parser = argparse.ArgumentParser(description="numSol vs anSol для D=±0.1")
    parser.add_argument("--output-dir", type=Path, default=DEFAULT_OUT_DIR)
    parser.add_argument("--no-run", action="store_true", help="Не запускать nsolv")
    args = parser.parse_args()
    out = args.output_dir.resolve()
    run_solver = not args.no_run

    case_pos = load_case("1blade_d01", run_solver=run_solver)
    case_neg = load_case("1blade_dneg01", run_solver=run_solver)

    save_single(case_pos, out / "damping_D0.1.png", y_lim=(-2.2, 2.2))
    save_single(case_neg, out / "damping_D-0.1.png", y_lim=(-22.0, 22.0))
    save_combined(
        [(case_pos, (-2.2, 2.2)), (case_neg, (-22.0, 22.0))],
        out / "damping_comparison.png",
    )


if __name__ == "__main__":
    main()
