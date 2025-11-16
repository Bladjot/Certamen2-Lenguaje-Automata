#!/usr/bin/env python3
"""Genera una visualización espacio-tiempo a partir del log JSON del simulador."""
import argparse
import json
import os
import sys

MPL_CACHE = os.path.join(os.path.dirname(__file__), ".matplotlib-cache")
os.makedirs(MPL_CACHE, exist_ok=True)
os.environ.setdefault("MPLCONFIGDIR", MPL_CACHE)

try:
    import matplotlib
    matplotlib.use("Agg")
    import matplotlib.pyplot as plt
    from matplotlib.patches import FancyArrowPatch
    from matplotlib.lines import Line2D
except ImportError as exc:  # pragma: no cover - depende del entorno
    print("matplotlib es requerido para generar la visualización:", exc, file=sys.stderr)
    sys.exit(1)
except Exception as exc:  # pragma: no cover - problemas de backend
    print("matplotlib no pudo inicializarse:", exc, file=sys.stderr)
    sys.exit(1)
SUMMARY_EVENTS = {
    "external_dispatched",
    "external_received",
    "external_processed",
    "internal_processed",
    "rollback_start",
    "rollback_end",
    "straggler_detected",
}


def load_entries(path: str):
    entries = []
    with open(path, "r", encoding="utf-8") as handler:
        for line_number, raw in enumerate(handler, start=1):
            raw = raw.strip()
            if not raw:
                continue
            try:
                entry = json.loads(raw)
            except json.JSONDecodeError as exc:  # pragma: no cover - depende del log
                print(f"Línea {line_number}: no se pudo parsear JSON ({exc})", file=sys.stderr)
                continue
            entries.append(entry)
    return entries


def filter_entries(entries, allowed):
    if not allowed:
        return entries, 0
    filtered = [entry for entry in entries if entry.get("event") in allowed]
    return filtered, len(entries) - len(filtered)


def draw_time_axis(ax, max_time):
    arrow = FancyArrowPatch(
        (-1.5, 0),
        (-1.5, max_time),
        arrowstyle="-|>",
        color="#0057b7",
        linewidth=2.5,
    )
    ax.add_patch(arrow)
    ax.text(-1.6, max_time / 2, "Tiempo", rotation=90, color="#0057b7", ha="center", va="center", fontsize=10)


def draw_lanes(ax, entities, max_time):
    positions = {}
    spacing = 1.0
    for idx, entity in enumerate(entities):
        x = idx * spacing
        positions[entity] = x
        ax.vlines(x, 0, max_time, colors="#111111", linewidth=2)
        ax.text(x, -0.7, entity, rotation=0, ha="center", va="top", fontsize=10)
    return positions


def add_legend(ax):
    handles = [
        Line2D([0], [0], color=EXTERNAL_COLOR, linewidth=2, marker=r"$\rightarrow$", markersize=12, label="Evento externo"),
        Line2D([0], [0], color=STRAGGLER_COLOR, linewidth=2, marker=r"$\rightarrow$", markersize=12, label="Straggler"),
        Line2D([0], [0], color=INTERNAL_COLOR, linewidth=2, marker=r"$\rightarrow$", markersize=12, label="Evento interno"),
        Line2D([0], [0], color=ROLLBACK_COLOR, linewidth=2, marker=r"$\rightarrow$", markersize=12, label="Rollback"),
        Line2D([0], [0], color="#1a9850", linewidth=3, label="Checkpoint"),
        Line2D([0], [0], marker="x", color="#e41a1c", linestyle="", markersize=8, label="Straggler detectado"),
    ]
    ax.legend(handles=handles, loc="upper left", bbox_to_anchor=(-0.6, 0.65))


EXTERNAL_COLOR = "#1f78b4"
STRAGGLER_COLOR = "#d73027"
INTERNAL_COLOR = "#66a61e"
ROLLBACK_COLOR = "#ff7f00"


def add_curved_arrow(ax, start, end, color, rad=0.2, style="-|>", linewidth=2):
    arrow = FancyArrowPatch(
        start,
        end,
        connectionstyle=f"arc3,rad={rad}",
        arrowstyle=style,
        color=color,
        linewidth=linewidth,
    )
    ax.add_patch(arrow)


def draw_event(ax, entry, positions, straggler_ids):
    event = entry.get("event")
    sim_time = entry.get("sim_time", 0)
    entity = entry.get("entity")
    if entity not in positions:
        return
    x = positions[entity]
    if event == "external_dispatched":
        target_id = entry.get("target_worker")
        target_entity = f"worker-{target_id}" if target_id is not None else None
        if target_entity in positions:
            start = (x, sim_time)
            end = (positions[target_entity], sim_time)
            event_id = entry.get("event_id")
            color = STRAGGLER_COLOR if event_id in straggler_ids else EXTERNAL_COLOR
            add_curved_arrow(ax, start, end, color, rad=0.15)
    elif event == "external_received":
        ax.hlines(sim_time, x - 0.25, x + 0.25, colors="#4daf4a", linewidth=4)
    elif event == "external_processed":
        ax.scatter(x, sim_time, color="#d95f02", s=35, marker="o", zorder=5)
    elif event == "internal_processed":
        prev = entry.get("details", {}).get("previous_lvt", sim_time)
        start = (x, prev)
        end = (x, sim_time)
        add_curved_arrow(ax, start, end, INTERNAL_COLOR, rad=0.5)
    elif event == "checkpoint_created":
        ax.hlines(sim_time, x - 0.2, x + 0.2, colors="#1a9850", linewidth=2)
    elif event == "straggler_detected":
        ax.plot([x - 0.25, x + 0.25], [sim_time - 0.25, sim_time + 0.25], color="#e41a1c", linewidth=2)
        ax.plot([x - 0.25, x + 0.25], [sim_time + 0.25, sim_time - 0.25], color="#e41a1c", linewidth=2)
    elif event == "rollback_start":
        rollback_to = entry.get("rollback_to", sim_time)
        rollback_from = entry.get("rollback_from", sim_time)
        start = (x, rollback_from)
        end = (x, rollback_to)
        add_curved_arrow(ax, start, end, ROLLBACK_COLOR, rad=-0.8)
    elif event == "rollback_end":
        ax.scatter(x, sim_time, color="#4daf4a", marker="s", s=40, zorder=5)


def plot(entries, output_path: str, show: bool, allowed_events=None):
    if not entries:
        print("El archivo de log está vacío, no hay nada que graficar", file=sys.stderr)
        return

    entries, removed = filter_entries(entries, allowed_events)
    if not entries:
        print("Los filtros seleccionados descartan todos los eventos", file=sys.stderr)
        return

    entities = sorted({entry.get("entity", "unknown") for entry in entries})
    max_time = max(entry.get("sim_time", 0) for entry in entries) + 2
    fig, ax = plt.subplots(figsize=(10, 6))

    draw_time_axis(ax, max_time)
    positions = draw_lanes(ax, entities, max_time)
    entries = sorted(entries, key=lambda item: (item.get("sim_time", 0), item.get("wall_time", "")))
    straggler_ids = {
        entry.get("event_id")
        for entry in entries
        if entry.get("event") == "straggler_detected" and entry.get("event_id") is not None
    }
    for entry in entries:
        draw_event(ax, entry, positions, straggler_ids)

    ax.set_xlim(-1, len(entities))
    ax.set_ylim(max_time, -1)
    ax.set_ylabel("Tiempo virtual (aumenta hacia abajo)")
    ax.set_xticks([])
    ax.set_title("Diagrama estilo espacio-tiempo (simplificado)")
    ax.spines["top"].set_visible(False)
    ax.spines["right"].set_visible(False)
    ax.spines["bottom"].set_visible(False)
    ax.spines["left"].set_visible(False)
    fig.tight_layout()
    add_legend(ax)
    fig.savefig(output_path, dpi=200)
    if show:
        plt.show()
    plt.close(fig)
    if removed:
        print(f"Eventos descartados por el filtro: {removed}")


def main():
    parser = argparse.ArgumentParser(description="Visualiza la ejecución del simulador desde un log JSONL.")
    parser.add_argument("log", help="Ruta del archivo JSONL generado por el simulador")
    parser.add_argument("-o", "--output", default="timeline.png", help="Imagen de salida (PNG)")
    parser.add_argument("--show", action="store_true", help="Muestra la figura en pantalla además de guardarla")
    parser.add_argument(
        "--mode",
        choices=["summary", "full"],
        default="summary",
        help="Modo de visualización: summary muestra sólo eventos clave, full muestra todo",
    )
    args = parser.parse_args()

    entries = load_entries(args.log)
    allowed = SUMMARY_EVENTS if args.mode == "summary" else None
    plot(entries, args.output, args.show, allowed)
    print(f"Visualización ({args.mode}) escrita en {args.output}")
if __name__ == "__main__":
    main()
