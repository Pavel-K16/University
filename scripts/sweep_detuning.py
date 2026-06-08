#!/usr/bin/env python3
"""
Перебор расстройки k между соседними лопатками (8blades2).
Чётные лопатки: k = 1 - delta; нечётные: k = 1 + delta; шаг delta = 0.005.
Остановка, когда epsilon > 30%.
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
from pathlib import Path

import matplotlib.pyplot as plt
import numpy as np

ROOT = Path(__file__).resolve().parent.parent
SCRIPTS = ROOT / "scripts"
CONF_PATH = ROOT / "internal" / "config" / "confs" / "8blades2.json"
CONF_BACKUP = ROOT / "internal" / "config" / "confs" / "8blades2.json.bak_sweep"
OUT_DIR = ROOT / "wolfram" / "detuning_sweep"
DATA_DIR = ROOT / "wolfram" / "paramsAndPoints"
PLOTS_DIR = OUT_DIR / "plots"
THESIS_DIR = ROOT / "doc" / "images"

HTML_SERIES_FIGSIZE = (10, 6)
HTML_LEGEND_RIGHT = 0.76
THESIS_DPI = 200

HUB_ID = 8
MASS = 2.0
K_STEP = 0.005
EPSILON_LIMIT = 30.0
PAIRS = [(0, 1), (2, 3), (4, 5), (6, 7)]
EVEN_BLADES = {0, 2, 4, 6}
ODD_BLADES = {1, 3, 5, 7}


def omega(k: float, m: float = MASS) -> float:
    return math.sqrt(k / m)


def epsilon_percent(k1: float, k2: float, m: float = MASS) -> float:
    w1 = omega(k1, m)
    w2 = omega(k2, m)
    return abs(w1 - w2) / ((w1 + w2) / 2.0) * 100.0


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


def run_simulation(bin_path: Path) -> None:
    env = os.environ.copy()
    env["CONFIG"] = "8blades2"
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
    xlim: tuple[float, float] | None = None,
    ylim: tuple[float, float] | None = None,
    xticks: list[float] | None = None,
    yticks: list[float] | None = None,
    y_series: list[np.ndarray] | None = None,
) -> None:
    """Оформление в стиле диплома (как finalize_html_series_plot в plot_points.py)."""
    ax = plt.gca()
    ax.set_xlabel("")
    ax.set_ylabel("")
    ax.grid(True, alpha=0.3)
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
        fontsize=12,
        clip_on=False,
    )
    ax.annotate(
        "ε, %",
        xy=(xmax, ymin),
        xycoords="data",
        xytext=(8, 0),
        textcoords="offset points",
        ha="left",
        va="center",
        fontsize=12,
        clip_on=False,
    )
    ax.legend(
        loc="upper left",
        bbox_to_anchor=(1.02, 1.0),
        frameon=True,
        borderaxespad=0.0,
        fontsize=10,
    )
    plt.gcf().subplots_adjust(left=0.08, right=HTML_LEGEND_RIGHT, top=0.96)
    plt.tight_layout(rect=(0.0, 0.0, HTML_LEGEND_RIGHT, 1.0))


def plot_detuning_series(
    ax: plt.Axes,
    eps: list[float],
    deltas: np.ndarray,
    *,
    color,
    label: str,
) -> None:
    ax.plot(
        eps,
        deltas,
        color=color,
        linewidth=1.2,
        marker="o",
        markersize=4,
        markerfacecolor=color,
        markeredgecolor=color,
        label=label,
    )


CROSSING_EPS_FONT_SIZE = 9
CROSSING_K_FONT_SIZE = 8
CROSSING_LINE_STEP_PX = 13
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


def plot_results(rows: list[dict]) -> dict[str, dict[str, float]]:
    PLOTS_DIR.mkdir(parents=True, exist_ok=True)
    THESIS_DIR.mkdir(parents=True, exist_ok=True)

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
    save_figure(THESIS_DIR / "detuning_delta_vs_epsilon_all_pairs.png", close=False)
    save_figure(PLOTS_DIR / "all_pairs.png")

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
        save_figure(THESIS_DIR / f"detuning_delta_vs_epsilon_pair_{a}_{b}.png", close=False)
        save_figure(PLOTS_DIR / f"pair_{a}_{b}.png")

    crossings_path = OUT_DIR / "zero_crossings.json"
    crossings_path.write_text(
        json.dumps(crossings, indent=2, ensure_ascii=False) + "\n",
        encoding="utf-8",
    )
    return crossings


def load_rows_from_csv(csv_path: Path) -> list[dict]:
    with csv_path.open(newline="", encoding="utf-8") as f:
        return list(csv.DictReader(f))


def main() -> int:
    parser = argparse.ArgumentParser(description="Перебор расстройки k (8blades2)")
    parser.add_argument(
        "--plot-only",
        action="store_true",
        help="Только построить графики из raw_results.csv (без пересчёта)",
    )
    args = parser.parse_args()

    OUT_DIR.mkdir(parents=True, exist_ok=True)
    PLOTS_DIR.mkdir(parents=True, exist_ok=True)

    csv_path = OUT_DIR / "raw_results.csv"
    if args.plot_only:
        if not csv_path.exists():
            print(f"Нет данных: {csv_path}", file=sys.stderr)
            return 1
        rows = load_rows_from_csv(csv_path)
        crossings = plot_results(rows)
        print(f"Графики: {THESIS_DIR}")
        print(f"Пересечение δ=0 (ε₀): {crossings}")
        return 0

    if not CONF_BACKUP.exists():
        CONF_BACKUP.write_text(CONF_PATH.read_text(encoding="utf-8"), encoding="utf-8")

    base_config = json.loads(CONF_BACKUP.read_text(encoding="utf-8"))
    deltas = generate_deltas()
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
            CONF_PATH.write_text(json.dumps(cfg, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")

            k_line = (
                f"step={step:03d} delta={delta:.3f} eps={eps:.4f}% "
                f"k: 0={k_map[0]:.3f} 1={k_map[1]:.3f} 2={k_map[2]:.3f} 3={k_map[3]:.3f} "
                f"4={k_map[4]:.3f} 5={k_map[5]:.3f} 6={k_map[6]:.3f} 7={k_map[7]:.3f}"
            )
            k_log_lines.append(k_line)
            print(k_line)

            run_simulation(bin_path)

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

            rows.append(row)

        with csv_path.open("w", newline="", encoding="utf-8") as f:
            writer = csv.DictWriter(f, fieldnames=fieldnames)
            writer.writeheader()
            writer.writerows(rows)

        k_log_path = OUT_DIR / "k_values.log"
        k_log_path.write_text("\n".join(k_log_lines) + "\n", encoding="utf-8")

        json_path = OUT_DIR / "raw_results.json"
        json_path.write_text(json.dumps(rows, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")

        crossings = plot_results(rows)
        print(f"\nГотово: {csv_path}")
        print(f"Графики (диплом): {THESIS_DIR}")
        print(f"Графики (preview): {PLOTS_DIR}")
        print(f"Пересечение δ=0 (ε₀): {crossings}")
    finally:
        CONF_PATH.write_text(CONF_BACKUP.read_text(encoding="utf-8"), encoding="utf-8")
        if bin_path.exists():
            bin_path.unlink()

    return 0


if __name__ == "__main__":
    sys.exit(main())
