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
except ImportError as exc:  # pragma: no cover - depende del entorno
    print("matplotlib es requerido para generar la visualización:", exc, file=sys.stderr)
    sys.exit(1)
except Exception as exc:  # pragma: no cover - problemas de backend
    print("matplotlib no pudo inicializarse:", exc, file=sys.stderr)
    sys.exit(1)


COLOR_MAP = {
    "external_dispatched": "#1b9e77",
    "external_received": "#7570b3",
    "external_processed": "#d95f02",
    "internal_processed": "#e7298a",
    "checkpoint_created": "#66a61e",
    "straggler_detected": "#e6ab02",
    "rollback_start": "#e41a1c",
    "rollback_end": "#4daf4a",
}

MARKERS = {
    "external_dispatched": "s",
    "external_received": "o",
    "external_processed": "^",
    "internal_processed": "v",
    "checkpoint_created": "x",
    "straggler_detected": "D",
    "rollback_start": "P",
    "rollback_end": "*",
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


def plot(entries, output_path: str, show: bool):
    if not entries:
        print("El archivo de log está vacío, no hay nada que graficar", file=sys.stderr)
        return

    entities = sorted({entry.get("entity", "unknown") for entry in entries})
    entity_index = {name: idx for idx, name in enumerate(entities)}
    height = max(4, len(entities) * 1.2)
    fig, ax = plt.subplots(figsize=(12, height))

    legend_handles = {}
    entries = sorted(entries, key=lambda item: (item.get("sim_time", 0), item.get("wall_time", "")))

    for entry in entries:
        entity = entry.get("entity", "unknown")
        y = entity_index.get(entity, 0)
        x = entry.get("sim_time", 0)
        event = entry.get("event", "")
        color = COLOR_MAP.get(event, "#333333")
        marker = MARKERS.get(event, "o")
        point = ax.scatter(x, y, color=color, marker=marker, s=50, alpha=0.85)
        if event not in legend_handles:
            legend_handles[event] = point

        if event == "rollback_start":
            from_time = entry.get("rollback_from")
            to_time = entry.get("rollback_to")
            if from_time is not None and to_time is not None:
                ax.annotate(
                    "",
                    xy=(to_time, y),
                    xytext=(from_time, y),
                    arrowprops=dict(arrowstyle="->", color="#e41a1c", linewidth=2),
                )

    ax.set_xlabel("Tiempo virtual")
    ax.set_ylabel("Entidad")
    ax.set_yticks(list(entity_index.values()))
    ax.set_yticklabels(entities)
    ax.set_title("Diagrama espacio-tiempo de la simulación")
    ax.grid(True, axis="x", linestyle="--", alpha=0.4)
    if legend_handles:
        ax.legend(legend_handles.values(), legend_handles.keys(), loc="best")
    fig.tight_layout()
    fig.savefig(output_path, dpi=200)
    if show:
        plt.show()
    plt.close(fig)


def main():
    parser = argparse.ArgumentParser(description="Visualiza la ejecución del simulador desde un log JSONL.")
    parser.add_argument("log", help="Ruta del archivo JSONL generado por el simulador")
    parser.add_argument("-o", "--output", default="timeline.png", help="Imagen de salida (PNG)")
    parser.add_argument("--show", action="store_true", help="Muestra la figura en pantalla además de guardarla")
    args = parser.parse_args()

    entries = load_entries(args.log)
    plot(entries, args.output, args.show)
    print(f"Visualización escrita en {args.output}")
if __name__ == "__main__":
    main()
