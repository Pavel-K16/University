#!/usr/bin/env python3
"""
Перебор расстройки k между соседними лопатками.
Чётные лопатки: k = 1 - delta; нечётные: k = 1 + delta; шаг delta = 0.005.
Остановка, когда epsilon > 30%.

Пример:
  python3 scripts/sweep_detuning.py --config 8blades2
  python3 scripts/sweep_detuning.py --config 8blades2 --plot-only

Графики: plots/all_blades.png (ε ↔ δ̄), plots/all_pairs.png (по парам), …
"""

from __future__ import annotations

import argparse
import csv
import json
import math
import os
import re
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path

import matplotlib.pyplot as plt
import numpy as np

ROOT = Path(__file__).resolve().parent.parent
SCRIPTS = ROOT / "scripts"
CONF_DIR = ROOT / "internal" / "config" / "confs"
DATA_DIR = ROOT / "wolfram" / "paramsAndPoints"
THESIS_DIR = ROOT / "doc" / "images"
DEFAULT_CONFIG = "8blades2"
SWEEP_ROOT = ROOT / "wolfram" / "detuning_sweep"
DELTA_MEAN_YLABEL = r"$\bar{\delta}$"

HTML_SERIES_FIGSIZE = (10, 6)
D_VS_DELTA_FIGSIZE = (13, 6)  # шире по X — деления d не слипаются
HTML_LEGEND_RIGHT = 0.76
THESIS_DPI = 200
THESIS_AXIS_LABEL_FONTSIZE = 21
THESIS_TICK_LABEL_FONTSIZE = 18
THESIS_LEGEND_FONTSIZE = 14

HUB_ID = 8
MASS = 2.0
K_STEP = 0.005
EPSILON_LIMIT = 30.0
PAIRS = [(0, 1), (2, 3), (4, 5), (6, 7)]
BLADE_IDS = list(range(8))
EVEN_BLADES = {0, 2, 4, 6}
ODD_BLADES = {1, 3, 5, 7}


@dataclass(frozen=True)
class SweepContext:
    config_name: str
    conf_path: Path
    conf_backup: Path
    out_dir: Path

    @property
    def plots_dir(self) -> Path:
        return self.out_dir / "plots"

    def thesis_name(self, stem: str) -> str:
        if self.config_name == DEFAULT_CONFIG:
            return f"detuning_{stem}.png"
        return f"detuning_{self.config_name}_{stem}.png"


def normalize_config_name(name: str) -> str:
    return name.strip().removesuffix(".json")


def resolve_out_dir(config_name: str) -> Path:
    """8blades2 — в wolfram/detuning_sweep/; остальные — в подпапку."""
    if config_name == DEFAULT_CONFIG:
        return SWEEP_ROOT
    return SWEEP_ROOT / config_name


def make_sweep_context(config_name: str) -> SweepContext:
    config_name = normalize_config_name(config_name)
    conf_path = CONF_DIR / f"{config_name}.json"
    if not conf_path.exists():
        raise FileNotFoundError(f"Конфиг не найден: {conf_path}")
    return SweepContext(
        config_name=config_name,
        conf_path=conf_path,
        conf_backup=CONF_DIR / f"{config_name}.json.bak_sweep",
        out_dir=resolve_out_dir(config_name),
    )


def omega(k: float, m: float = MASS) -> float:
    return math.sqrt(k / m)


def epsilon_percent(k1: float, k2: float, m: float = MASS) -> float:
    w1 = omega(k1, m)
    w2 = omega(k2, m)
    return abs(w1 - w2) / ((w1 + w2) / 2.0) * 100.0


def natural_frequencies_from_row(row: dict, m: float = MASS) -> np.ndarray:
    """ω₀ᵢ = √(kᵢ/m) по аналитической формуле из конфига."""
    return np.array([omega(float(row[f"k_{i}"]), m) for i in BLADE_IDS], dtype=float)


def compute_d_from_row(row: dict, m: float = MASS) -> tuple[float, float, float]:
    """d = RMSE(ω₀ᵢ) / Ave(ω₀ᵢ); возвращает (d, rmse, ave)."""
    omegas = natural_frequencies_from_row(row, m)
    ave = float(np.mean(omegas))
    if ave <= 0:
        return 0.0, 0.0, ave
    rmse = float(np.sqrt(np.mean((omegas - ave) ** 2)))
    return rmse / ave, rmse, ave


def mean_delta_all_blades(row: dict) -> float:
    """Средний декремент δ̄ по всем подвижным лопаткам."""
    return float(np.mean([float(row[f"delta_node_{i}"]) for i in BLADE_IDS]))


def enrich_row(row: dict) -> dict:
    out = dict(row)
    d, rmse, ave = compute_d_from_row(out)
    out["omega_avg"] = round(ave, 9)
    out["omega_rmse"] = round(rmse, 9)
    out["d"] = round(d, 9)
    out["delta_mean"] = mean_delta_all_blades(out)
    return out


def enrich_rows(rows: list[dict]) -> list[dict]:
    return [enrich_row(r) for r in rows]


def hub_k_values(delta: float) -> dict[int, float]:
    """k для рёбер лопатка -> хаб (узел 8)."""
    return {b: 1.0 - delta if b in EVEN_BLADES else 1.0 + delta for b in range(8)}


def apply_k_to_config(config: dict, k_by_blade: dict[int, float]) -> dict:
    cfg = json.loads(json.dumps(config))
    for edge in cfg["edges"]:
        if edge.get("to") == HUB_ID and edge.get("from") in k_by_blade:
            edge["k"] = round(k_by_blade[edge["from"]], 6)
    return cfg


def mean_delta_from_file(path: Path) -> float | None:
    if not path.exists():
        return None
    vals: list[float] = []
    for line in path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line:
            continue
        parts = line.split()
        if len(parts) >= 2:
            vals.append(float(parts[1]))
    if not vals:
        return None
    return sum(vals) / len(vals)


def parse_node_deltas_from_log(log_path: Path) -> dict[int, float]:
    """Запасной парсер: delta из decrement_details.log."""
    if not log_path.exists():
        return {}
    text = log_path.read_text(encoding="utf-8")
    out: dict[int, float] = {}
    for m in re.finditer(r"Node (\d+): xEq=.*?, delta=([-\d.]+),", text):
        out[int(m.group(1))] = float(m.group(2))
    return out


def build_binary() -> Path:
    bin_path = SCRIPTS / "sweep_nsolv_bin"
    subprocess.run(
        ["go", "build", "-o", str(bin_path), str(ROOT / "cmd" / "nsolv.go")],
        cwd=SCRIPTS,
        check=True,
    )
    return bin_path


def run_simulation(bin_path: Path, ctx: SweepContext) -> None:
    env = os.environ.copy()
    env["CONFIG"] = ctx.config_name
    subprocess.run(
        [str(bin_path)],
        cwd=SCRIPTS,
        env=env,
        check=True,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )


def generate_deltas() -> list[float]:
    deltas: list[float] = []
    d = 0.0
    while True:
        k_lo = 1.0 - d
        k_hi = 1.0 + d
        eps = epsilon_percent(k_lo, k_hi)
        if eps > EPSILON_LIMIT:
            break
        deltas.append(round(d, 6))
        d += K_STEP
    return deltas


def interpolate_k_at_eps(rows: list[dict], eps_target: float, blade_id: int) -> float:
    eps_arr = np.array([float(r["epsilon_pct"]) for r in rows], dtype=float)
    k_arr = np.array([float(r[f"k_{blade_id}"]) for r in rows], dtype=float)
    return float(np.interp(eps_target, eps_arr, k_arr))


def interpolate_field_at_eps(rows: list[dict], eps_target: float, field: str) -> float:
    eps_arr = np.array([float(r["epsilon_pct"]) for r in rows], dtype=float)
    vals = np.array([float(r[field]) for r in rows], dtype=float)
    return float(np.interp(eps_target, eps_arr, vals))


def find_zero_crossing_eps(eps: list[float], deltas: list[float]) -> float | None:
    """Линейная интерполяция точки пересечения δ=0."""
    for i in range(1, len(deltas)):
        y0, y1 = deltas[i - 1], deltas[i]
        if y0 == 0.0:
            return eps[i - 1]
        if y0 * y1 < 0.0:
            e0, e1 = eps[i - 1], eps[i]
            t = -y0 / (y1 - y0)
            return e0 + t * (e1 - e0)
    return None


def auto_plot_ylim(y_series: list[np.ndarray], pad_frac: float = 0.08) -> tuple[float, float] | None:
    parts = [y[np.isfinite(y)] for y in y_series if y.size]
    if not parts:
        return None
    all_y = np.concatenate(parts)
    lo, hi = float(np.min(all_y)), float(np.max(all_y))
    if lo == hi:
        pad = max(abs(lo) * pad_frac, 0.01)
        return lo - pad, hi + pad
    pad = (hi - lo) * pad_frac
    return lo - pad, hi + pad


def finalize_thesis_detuning_plot(
    *,
    ylabel: str = "δ",
    x_label: str = "ε, %",
    xlim: tuple[float, float] | None = None,
    ylim: tuple[float, float] | None = None,
    xticks: list[float] | None = None,
    yticks: list[float] | None = None,
    y_series: list[np.ndarray] | None = None,
    show_legend: bool = True,
) -> None:
    """Оформление в стиле диплома (как finalize_html_series_plot в plot_points.py)."""
    ax = plt.gca()
    ax.set_xlabel("")
    ax.set_ylabel("")
    ax.grid(True, alpha=0.3)
    ax.tick_params(axis="both", labelsize=THESIS_TICK_LABEL_FONTSIZE)
    if xlim is not None:
        ax.set_xlim(*xlim)
    if ylim is not None:
        ax.set_ylim(*ylim)
    elif y_series:
        auto_ylim = auto_plot_ylim(y_series)
        if auto_ylim is not None:
            ax.set_ylim(*auto_ylim)
    if xticks is not None:
        ax.set_xticks(xticks)
    if yticks is not None:
        ax.set_yticks(yticks)
    ymin = ax.get_ylim()[0]
    xmax = ax.get_xlim()[1]
    ax.text(
        0.0,
        1.02,
        ylabel,
        transform=ax.transAxes,
        ha="left",
        va="bottom",
        fontsize=THESIS_AXIS_LABEL_FONTSIZE,
        clip_on=False,
    )
    ax.annotate(
        x_label,
        xy=(xmax, ymin),
        xycoords="data",
        xytext=(8, 0),
        textcoords="offset points",
        ha="left",
        va="center",
        fontsize=THESIS_AXIS_LABEL_FONTSIZE,
        clip_on=False,
    )
    right_margin = HTML_LEGEND_RIGHT if show_legend else 0.92
    if show_legend:
        ax.legend(
            loc="upper left",
            bbox_to_anchor=(1.02, 1.0),
            frameon=True,
            borderaxespad=0.0,
            fontsize=THESIS_LEGEND_FONTSIZE,
        )
    plt.gcf().subplots_adjust(left=0.08, right=right_margin, top=0.96)
    plt.tight_layout(rect=(0.0, 0.0, right_margin, 1.0))


def plot_detuning_series(
    ax: plt.Axes,
    x: list[float] | np.ndarray,
    y: np.ndarray,
    *,
    color,
    label: str | None = None,
) -> None:
    ax.plot(
        x,
        y,
        color=color,
        linewidth=1.2,
        marker="o",
        markersize=4,
        markerfacecolor=color,
        markeredgecolor=color,
        label=label,
    )


def draw_d_zero_crossing_marker(
    ax: plt.Axes,
    d_zero: float,
    *,
    color: str = "#1f77b4",
) -> None:
    ymin, _ = ax.get_ylim()
    ax.axvline(d_zero, color=color, linestyle="--", linewidth=1.2, alpha=0.85, zorder=3)
    ax.axhline(0.0, color="gray", linestyle=":", linewidth=0.8, alpha=0.6, zorder=1)
    ax.plot([d_zero], [0.0], "o", color=color, markersize=6, zorder=4)
    ax.annotate(
        f"d₀ = {d_zero:.4f}",
        xy=(d_zero, ymin),
        xycoords="data",
        xytext=(6, 6),
        textcoords="offset points",
        ha="left",
        va="bottom",
        fontsize=CROSSING_EPS_FONT_SIZE,
        color=color,
        clip_on=False,
    )


def nice_d_xticks(d_values: np.ndarray, count: int = 7) -> list[float]:
    d_max = float(np.max(d_values))
    if d_max <= 0:
        return [0.0]
    # Для презентации: крупный шаг, чтобы подписи d не слипались
    for step in (0.05, 0.04, 0.03, 0.025, 0.02, 0.015):
        ticks = []
        v = 0.0
        while v <= d_max + step * 0.01:
            ticks.append(round(v, 4))
            v += step
        if len(ticks) <= count + 1:
            return ticks
    step = d_max / max(count - 1, 1)
    magnitude = 10 ** math.floor(math.log10(step)) if step > 0 else 0.001
    step = max(magnitude, math.ceil(step / magnitude) * magnitude / 2)
    ticks = []
    v = 0.0
    while v <= d_max + step * 0.01:
        ticks.append(round(v, 6))
        v += step
    return ticks


def plot_d_vs_mean_delta(rows: list[dict], ctx: SweepContext) -> float | None:
    """График d = RMSE(ω₀ᵢ)/Ave(ω₀ᵢ) от среднего декремента δ̄."""
    d_vals = np.array([float(r["d"]) for r in rows], dtype=float)
    delta_mean = np.array([float(r["delta_mean"]) for r in rows], dtype=float)
    color = "#1f77b4"

    d0 = find_zero_crossing_eps(d_vals.tolist(), delta_mean.tolist())
    xlim = (0.0, float(np.max(d_vals)) * 1.02)
    xticks = nice_d_xticks(d_vals, count=6)
    xticklabels = [f"{t:.2f}" for t in xticks]

    fig, ax = plt.subplots(figsize=D_VS_DELTA_FIGSIZE)
    plot_detuning_series(ax, d_vals, delta_mean, color=color, label=None)
    finalize_thesis_detuning_plot(
        ylabel=DELTA_MEAN_YLABEL,
        x_label="d",
        xlim=xlim,
        ylim=auto_plot_ylim([delta_mean]),
        xticks=xticks,
        y_series=[delta_mean],
        show_legend=False,
    )
    ax.set_xticklabels(xticklabels)
    if d0 is not None:
        draw_d_zero_crossing_marker(ax, d0, color=color)
    save_figure(THESIS_DIR / ctx.thesis_name("d_vs_delta_mean"), close=False)
    save_figure(ctx.plots_dir / "d_vs_delta_mean.png")
    return d0


def plot_delta_vs_epsilon_all_blades(rows: list[dict], ctx: SweepContext) -> float | None:
    """ε ↔ средний декремент δ̄ по всем 8 лопаткам (одна кривая)."""
    eps = [float(r["epsilon_pct"]) for r in rows]
    delta_mean = np.array([float(r["delta_mean"]) for r in rows], dtype=float)
    color = "#1f77b4"
    eps0 = find_zero_crossing_eps(eps, delta_mean.tolist())
    xlim = (0.0, EPSILON_LIMIT)
    xticks = list(range(0, 31, 5))

    fig, ax = plt.subplots(figsize=HTML_SERIES_FIGSIZE)
    plot_detuning_series(ax, eps, delta_mean, color=color, label=None)
    finalize_thesis_detuning_plot(
        ylabel=DELTA_MEAN_YLABEL,
        xlim=xlim,
        ylim=auto_plot_ylim([delta_mean]),
        xticks=xticks,
        y_series=[delta_mean],
        show_legend=False,
    )
    if eps0 is not None:
        k_even = interpolate_field_at_eps(rows, eps0, "k_even")
        k_odd = interpolate_field_at_eps(rows, eps0, "k_odd")
        draw_zero_crossing_marker(
            ax,
            eps0,
            color=color,
            blade_a=0,
            blade_b=1,
            k_a=k_even,
            k_b=k_odd,
        )
    save_figure(THESIS_DIR / ctx.thesis_name("delta_vs_epsilon_all_blades"), close=False)
    save_figure(ctx.plots_dir / "all_blades.png")
    return eps0


CROSSING_EPS_FONT_SIZE = 13
CROSSING_K_FONT_SIZE = 12
CROSSING_LINE_STEP_PX = 16
_SUBSCRIPTS = "₀₁₂₃₄₅₆₇₈₉"


def blade_subscript(blade_id: int) -> str:
    return "".join(_SUBSCRIPTS[int(ch)] for ch in str(blade_id))


def crossing_label_lines(
    eps_zero: float,
    *,
    blade_a: int | None = None,
    blade_b: int | None = None,
    k_a: float | None = None,
    k_b: float | None = None,
    pair_tag: str | None = None,
) -> list[tuple[str, int]]:
    """Столбик снизу вверх: ε₀ у оси, выше — k с нижними индексами."""
    lines: list[tuple[str, int]] = []
    if pair_tag:
        lines.append((f"ε₀({pair_tag}) = {eps_zero:.2f}%", CROSSING_EPS_FONT_SIZE))
    else:
        lines.append((f"ε₀ = {eps_zero:.2f}%", CROSSING_EPS_FONT_SIZE))
    if blade_a is not None and k_a is not None:
        lines.append((f"k{blade_subscript(blade_a)} = {k_a:.3f}", CROSSING_K_FONT_SIZE))
    if blade_b is not None and k_b is not None:
        lines.append((f"k{blade_subscript(blade_b)} = {k_b:.3f}", CROSSING_K_FONT_SIZE))
    return lines


def draw_zero_crossing_marker(
    ax: plt.Axes,
    eps_zero: float,
    *,
    color: str = "#c0392b",
    blade_a: int | None = None,
    blade_b: int | None = None,
    k_a: float | None = None,
    k_b: float | None = None,
    pair_tag: str | None = None,
    y_offset_px: int = 6,
) -> None:
    ymin, _ = ax.get_ylim()
    ax.axvline(eps_zero, color=color, linestyle="--", linewidth=1.2, alpha=0.85, zorder=3)
    ax.axhline(0.0, color="gray", linestyle=":", linewidth=0.8, alpha=0.6, zorder=1)
    ax.plot([eps_zero], [0.0], "o", color=color, markersize=6, zorder=4)
    for line_idx, (text, font_size) in enumerate(
        crossing_label_lines(
            eps_zero,
            blade_a=blade_a,
            blade_b=blade_b,
            k_a=k_a,
            k_b=k_b,
            pair_tag=pair_tag,
        )
    ):
        ax.annotate(
            text,
            xy=(eps_zero, ymin),
            xycoords="data",
            xytext=(6, y_offset_px + line_idx * CROSSING_LINE_STEP_PX),
            textcoords="offset points",
            ha="left",
            va="bottom",
            fontsize=font_size,
            color=color,
            clip_on=False,
        )


def save_figure(path: Path, *, close: bool = True) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    plt.savefig(path, dpi=THESIS_DPI, bbox_inches="tight", pad_inches=0.06)
    if close:
        plt.close()


def crossing_info_at_eps0(rows: list[dict], a: int, b: int, eps0: float) -> dict[str, float]:
    return {
        "epsilon_pct": eps0,
        f"k_{a}": interpolate_k_at_eps(rows, eps0, a),
        f"k_{b}": interpolate_k_at_eps(rows, eps0, b),
    }


def plot_results(rows: list[dict], ctx: SweepContext) -> tuple[dict[str, dict[str, float]], float | None, float | None]:
    ctx.plots_dir.mkdir(parents=True, exist_ok=True)
    THESIS_DIR.mkdir(parents=True, exist_ok=True)

    rows = enrich_rows(rows)
    save_enriched_results(rows, ctx)
    d0 = plot_d_vs_mean_delta(rows, ctx)
    eps0_all = plot_delta_vs_epsilon_all_blades(rows, ctx)

    colors = plt.cm.tab10(np.linspace(0, 0.4, len(PAIRS)))
    eps = [float(r["epsilon_pct"]) for r in rows]
    xlim = (0.0, EPSILON_LIMIT)
    xticks = list(range(0, 31, 5))
    crossings: dict[str, dict[str, float]] = {}

    series_data: list[tuple[int, int, np.ndarray, float | None, float | None, float | None]] = []
    for a, b in PAIRS:
        col = f"delta_pair_{a}_{b}"
        deltas = np.array([float(r[col]) for r in rows], dtype=float)
        eps0 = find_zero_crossing_eps(eps, deltas.tolist())
        k_a = k_b = None
        key = f"{a}_{b}"
        if eps0 is not None:
            k_a = interpolate_k_at_eps(rows, eps0, a)
            k_b = interpolate_k_at_eps(rows, eps0, b)
            crossings[key] = crossing_info_at_eps0(rows, a, b, eps0)
        series_data.append((a, b, deltas, eps0, k_a, k_b))

    y_all = [d for _, _, d, _, _, _ in series_data]
    ylim = auto_plot_ylim(y_all)
    if ylim is not None:
        lo, hi = ylim
        ylim = (lo, hi)

    # --- все пары: doc/images + preview ---
    fig, ax = plt.subplots(figsize=HTML_SERIES_FIGSIZE)
    for idx, (a, b, deltas, _, _, _) in enumerate(series_data):
        plot_detuning_series(ax, eps, deltas, color=colors[idx], label=f"пара {a}–{b}")
    finalize_thesis_detuning_plot(
        ylabel="δ",
        xlim=xlim,
        ylim=ylim,
        xticks=xticks,
        y_series=y_all,
    )
    for idx, (a, b, _, eps0, k_a, k_b) in enumerate(series_data):
        if eps0 is not None and k_a is not None and k_b is not None:
            draw_zero_crossing_marker(
                ax,
                eps0,
                color=colors[idx],
                pair_tag=f"{a}-{b}",
                blade_a=a,
                blade_b=b,
                k_a=k_a,
                k_b=k_b,
                y_offset_px=6 + idx * 48,
            )
    save_figure(THESIS_DIR / ctx.thesis_name("delta_vs_epsilon_all_pairs"), close=False)
    save_figure(ctx.plots_dir / "all_pairs.png")

    # --- по одной паре ---
    for idx, (a, b, deltas, eps0, k_a, k_b) in enumerate(series_data):
        fig, ax = plt.subplots(figsize=HTML_SERIES_FIGSIZE)
        plot_detuning_series(ax, eps, deltas, color=colors[idx], label=f"пара {a}–{b}")
        finalize_thesis_detuning_plot(
            ylabel="δ",
            xlim=xlim,
            ylim=auto_plot_ylim([deltas]),
            xticks=xticks,
            y_series=[deltas],
        )
        if eps0 is not None and k_a is not None and k_b is not None:
            draw_zero_crossing_marker(
                ax,
                eps0,
                color=colors[idx],
                blade_a=a,
                blade_b=b,
                k_a=k_a,
                k_b=k_b,
            )
        save_figure(THESIS_DIR / ctx.thesis_name(f"delta_vs_epsilon_pair_{a}_{b}"), close=False)
        save_figure(ctx.plots_dir / f"pair_{a}_{b}.png")

    crossings_path = ctx.out_dir / "zero_crossings.json"
    all_blades_crossing: dict[str, float] | None = None
    if eps0_all is not None:
        all_blades_crossing = {
            "epsilon_pct": eps0_all,
            "k_even": interpolate_field_at_eps(rows, eps0_all, "k_even"),
            "k_odd": interpolate_field_at_eps(rows, eps0_all, "k_odd"),
        }
    summary = {
        "config": ctx.config_name,
        "epsilon_pairs": crossings,
        "epsilon_all_blades": all_blades_crossing,
        "d_at_delta_mean_zero": d0,
    }
    crossings_path.write_text(
        json.dumps(summary, indent=2, ensure_ascii=False) + "\n",
        encoding="utf-8",
    )
    return crossings, d0, eps0_all


def enriched_fieldnames(rows: list[dict]) -> list[str]:
    preferred = [
        "step", "delta_k", "epsilon_pct", "k_even", "k_odd",
        "d", "omega_avg", "omega_rmse", "delta_mean",
    ]
    for a, b in PAIRS:
        preferred.extend([
            f"k_{a}", f"k_{b}",
            f"delta_node_{a}", f"delta_node_{b}",
            f"delta_pair_{a}_{b}",
        ])
    if not rows:
        return preferred
    existing = list(rows[0].keys())
    return [name for name in preferred if name in existing] + [
        name for name in existing if name not in preferred
    ]


def save_enriched_results(rows: list[dict], ctx: SweepContext) -> None:
    ctx.out_dir.mkdir(parents=True, exist_ok=True)
    csv_path = ctx.out_dir / "raw_results.csv"
    json_path = ctx.out_dir / "raw_results.json"
    fieldnames = enriched_fieldnames(rows)
    with csv_path.open("w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(rows)
    json_path.write_text(json.dumps(rows, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")


def load_rows_from_csv(csv_path: Path) -> list[dict]:
    with csv_path.open(newline="", encoding="utf-8") as f:
        return list(csv.DictReader(f))


def main() -> int:
    parser = argparse.ArgumentParser(description="Перебор расстройки k для конфига с 8 лопатками")
    parser.add_argument(
        "--config",
        default=DEFAULT_CONFIG,
        help=f"Имя конфига без .json (по умолчанию: {DEFAULT_CONFIG})",
    )
    parser.add_argument(
        "--plot-only",
        action="store_true",
        help="Только построить графики из raw_results.csv (без пересчёта)",
    )
    args = parser.parse_args()

    try:
        ctx = make_sweep_context(args.config)
    except FileNotFoundError as exc:
        print(exc, file=sys.stderr)
        return 1

    ctx.out_dir.mkdir(parents=True, exist_ok=True)
    ctx.plots_dir.mkdir(parents=True, exist_ok=True)

    csv_path = ctx.out_dir / "raw_results.csv"
    if args.plot_only:
        if not csv_path.exists():
            print(f"Нет данных: {csv_path}", file=sys.stderr)
            return 1
        rows = load_rows_from_csv(csv_path)
        crossings, d0, eps0_all = plot_results(rows, ctx)
        print(f"Конфиг: {ctx.config_name}")
        print(f"Данные: {ctx.out_dir}")
        print(f"Графики (диплом): {THESIS_DIR}")
        print(f"Пересечение δ=0 (ε₀) по парам: {crossings}")
        print(f"Пересечение δ̄=0 (ε₀): {eps0_all}")
        print(f"Пересечение δ̄=0 (d₀): {d0}")
        return 0

    if not ctx.conf_backup.exists():
        ctx.conf_backup.write_text(ctx.conf_path.read_text(encoding="utf-8"), encoding="utf-8")

    base_config = json.loads(ctx.conf_backup.read_text(encoding="utf-8"))
    deltas = generate_deltas()
    print(f"Конфиг: {ctx.config_name}")
    print(f"Точек перебора: {len(deltas)} (epsilon <= {EPSILON_LIMIT}%)")

    bin_path = build_binary()
    rows: list[dict] = []
    k_log_lines: list[str] = []

    fieldnames = [
        "step",
        "delta_k",
        "epsilon_pct",
        "k_even",
        "k_odd",
        "d",
        "omega_avg",
        "omega_rmse",
        "delta_mean",
    ]
    for a, b in PAIRS:
        fieldnames.extend([
            f"k_{a}",
            f"k_{b}",
            f"delta_node_{a}",
            f"delta_node_{b}",
            f"delta_pair_{a}_{b}",
        ])

    try:
        for step, delta in enumerate(deltas):
            k_map = hub_k_values(delta)
            k_lo = k_map[0]
            k_hi = k_map[1]
            eps = epsilon_percent(k_lo, k_hi)

            cfg = apply_k_to_config(base_config, k_map)
            ctx.conf_path.write_text(json.dumps(cfg, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")

            k_line = (
                f"step={step:03d} delta={delta:.3f} eps={eps:.4f}% "
                f"k: 0={k_map[0]:.3f} 1={k_map[1]:.3f} 2={k_map[2]:.3f} 3={k_map[3]:.3f} "
                f"4={k_map[4]:.3f} 5={k_map[5]:.3f} 6={k_map[6]:.3f} 7={k_map[7]:.3f}"
            )
            k_log_lines.append(k_line)
            print(k_line)

            run_simulation(bin_path, ctx)

            node_deltas: dict[int, float | None] = {}
            for n in range(8):
                node_deltas[n] = mean_delta_from_file(DATA_DIR / f"decrement_points{n}.txt")

            log_deltas = parse_node_deltas_from_log(DATA_DIR / "decrement_details.log")
            for n in range(8):
                if node_deltas[n] is None and n in log_deltas:
                    node_deltas[n] = log_deltas[n]

            row: dict = {
                "step": step,
                "delta_k": delta,
                "epsilon_pct": round(eps, 6),
                "k_even": k_lo,
                "k_odd": k_hi,
            }
            for a, b in PAIRS:
                da = node_deltas.get(a)
                db = node_deltas.get(b)
                row[f"k_{a}"] = k_map[a]
                row[f"k_{b}"] = k_map[b]
                row[f"delta_node_{a}"] = da if da is not None else ""
                row[f"delta_node_{b}"] = db if db is not None else ""
                if da is not None and db is not None:
                    row[f"delta_pair_{a}_{b}"] = (da + db) / 2.0
                else:
                    row[f"delta_pair_{a}_{b}"] = ""

            rows.append(enrich_row(row))

        with csv_path.open("w", newline="", encoding="utf-8") as f:
            writer = csv.DictWriter(f, fieldnames=fieldnames)
            writer.writeheader()
            writer.writerows(rows)

        k_log_path = ctx.out_dir / "k_values.log"
        k_log_path.write_text("\n".join(k_log_lines) + "\n", encoding="utf-8")

        json_path = ctx.out_dir / "raw_results.json"
        json_path.write_text(json.dumps(rows, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")

        crossings, d0, eps0_all = plot_results(rows, ctx)
        print(f"\nГотово: {csv_path}")
        print(f"Графики (диплом): {THESIS_DIR}")
        print(f"Графики (preview): {ctx.plots_dir}")
        print(f"Пересечение δ=0 (ε₀) по парам: {crossings}")
        print(f"Пересечение δ̄=0 (ε₀): {eps0_all}")
        print(f"Пересечение δ̄=0 (d₀): {d0}")
    finally:
        ctx.conf_path.write_text(ctx.conf_backup.read_text(encoding="utf-8"), encoding="utf-8")
        if bin_path.exists():
            bin_path.unlink()

    return 0


if __name__ == "__main__":
    sys.exit(main())
