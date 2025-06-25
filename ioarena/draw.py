from __future__ import annotations

import argparse
import glob
import re
from pathlib import Path
from typing import Any, Dict, List

import matplotlib.pyplot as plt
import pandas as pd


_ENGINE_RE = re.compile(r"engine\s*=\s*(\S+)")
_MODE_RE = re.compile(r"durability\s*=\s*(\S+)")
_THROUGHPUT_RE = re.compile(
    r"throughput:\s+([0-9.]+)\s*([kmg]?)\s*ops/s",
    re.I,
)
_LATENCY_RE = re.compile(
    r"avg latency:\s+([0-9.]+)\s*(ns|us|µs|ms|s)/op",
    re.I,
)
_CPU_RE = re.compile(
    r"cpu:\s*user\s+([0-9.]+)(?:\s*,\s*system\s+([0-9.]+))?",
    re.I,
)
_SPACE_RE = re.compile(r"space:\s*disk\s+([0-9.]+)", re.I)

_MULTIPLIER = {
    "": 1,
    "K": 1_000,
    "M": 1_000_000,
    "G": 1_000_000_000,
    "k": 1_000,
    "m": 1_000_000,
    "g": 1_000_000_000,
}
_LATENCY_SCALE_TO_US = {
    "ns": 1e-3,
    "us": 1,
    "µs": 1,
    "ms": 1_000,
    "s": 1_000_000,
}




def _parse_file(path: str | Path) -> Dict[str, Any]:
    metrics: Dict[str, Any] = {}
    with open(path, "r", encoding="utf-8", errors="ignore") as fh:
        for line in fh:
            if (m := _ENGINE_RE.search(line)) and "engine" not in metrics:
                metrics["engine"] = m.group(1)
            if (m := _MODE_RE.search(line)) and "mode" not in metrics:
                metrics["mode"] = m.group(1)
            if (m := _THROUGHPUT_RE.search(line)) and "throughput_ops" not in metrics:
                value, prefix = float(m.group(1)), m.group(2).upper()
                metrics["throughput_ops"] = value * _MULTIPLIER.get(prefix, 1)
            if (m := _LATENCY_RE.search(line)) and "avg_latency_us" not in metrics:
                val, unit = float(m.group(1)), m.group(2).lower()
                metrics["avg_latency_us"] = val * _LATENCY_SCALE_TO_US.get(unit, 1)
            if (m := _CPU_RE.search(line)) and "cpu_user" not in metrics:
                metrics["cpu_user"] = float(m.group(1))
                if m.group(2):
                    metrics["cpu_system"] = float(m.group(2))
                else:
                    metrics["cpu_system"] = 0.0
            if (m := _SPACE_RE.search(line)) and "disk_mb" not in metrics:
                metrics["disk_mb"] = float(m.group(1))

    metrics["file"] = str(path)
    return metrics


def _collect(directory: str | Path) -> pd.DataFrame:
    rows: List[Dict[str, Any]] = []
    for txt in glob.glob(str(Path(directory) / "*.txt")):
        row = _parse_file(txt)
        if row.get("engine") and row.get("mode"):
            rows.append(row)
    return pd.DataFrame(rows)


def _grouped_bar(df: pd.DataFrame, value: str, ylabel: str, title: str, out: Path) -> None:
    engines = sorted(df["engine"].unique())
    modes = sorted(df["mode"].unique())
    x = range(len(engines))
    width = 0.8 / max(len(modes), 1)

    fig, ax = plt.subplots(figsize=(8, 5))

    for idx, mode in enumerate(modes):
        subset = (
            df[df["mode"] == mode]
            .set_index("engine")
            .reindex(engines)
            .fillna(0)
        )
        ax.bar([i + idx * width for i in x], subset[value], width, label=mode)

    ax.set_xticks([i + width * (len(modes) - 1) / 2 for i in x])
    ax.set_xticklabels(engines, rotation=45, ha="right")
    ax.set_ylabel(ylabel)
    ax.set_title(title)
    if len(modes) > 1:
        ax.legend(title="Durability")
    fig.tight_layout()
    fig.savefig(out, dpi=150)
    plt.close(fig)


def _cpu_stacked(df: pd.DataFrame, out: Path) -> None:
    engines = sorted(df["engine"].unique())
    modes = sorted(df["mode"].unique())
    x = range(len(engines))
    width = 0.8 / max(len(modes), 1)

    fig, ax = plt.subplots(figsize=(8, 5))

    for idx, mode in enumerate(modes):
        subset = (
            df[df["mode"] == mode]
            .set_index("engine")
            .reindex(engines)
            .fillna(0)
        )
        users = subset["cpu_user"]
        systems = subset["cpu_system"]
        xpos = [i + idx * width for i in x]
        ax.bar(xpos, users, width, label=f"{mode} user")
        ax.bar(xpos, systems, width, bottom=users, label=f"{mode} system", hatch="//")

    ax.set_xticks([i + width * (len(modes) - 1) / 2 for i in x])
    ax.set_xticklabels(engines, rotation=45, ha="right")
    ax.set_ylabel("CPU time (seconds)")
    ax.set_title("CPU usage breakdown (user + system)")
    ax.legend(ncol=2)
    fig.tight_layout()
    fig.savefig(out, dpi=150)
    plt.close(fig)


def main() -> None:
    p = argparse.ArgumentParser(description="Draw IOarena summary graphs (v2)")
    p.add_argument("-d", "--dir", default="new_results", help="Directory with *.txt output files")
    p.add_argument("-o", "--out", default="plots", help="Directory to save PNG files")
    args = p.parse_args()

    out_dir = Path(args.out)
    out_dir.mkdir(parents=True, exist_ok=True)

    df = _collect(Path(args.dir))

    required = ["throughput_ops", "avg_latency_us", "cpu_user", "cpu_system", "disk_mb"]
    df = df.dropna(subset=required)
    if df.empty:
        raise SystemExit("❌ No complete metrics found. Check the input directory.")

    _grouped_bar(df, "throughput_ops", "Throughput (ops/s)", "IOarena Throughput", out_dir / "throughput.png")
    _grouped_bar(df, "avg_latency_us", "Average latency (µs)", "Average Latency", out_dir / "latency.png")
    _cpu_stacked(df, out_dir / "cpu.png")
    _grouped_bar(df, "disk_mb", "Disk usage (MB)", "Disk Footprint", out_dir / "disk.png")

    print(f"✅ Plots saved to: {out_dir.resolve()}")


if __name__ == "__main__":
    main()