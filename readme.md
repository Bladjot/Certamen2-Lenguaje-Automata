# Simulación de Scheduler/Workers con Rollbacks Optimistas

Este proyecto implementa un simulador multi-hilo que modela un scheduler y un conjunto de workers
que procesan eventos externos e internos bajo un protocolo optimista. El objetivo es reproducir
violaciones de causalidad (stragglers), ejecutar rollbacks usando checkpoints y producir trazas que
puedan ser visualizadas offline.

## Componentes Principales

- **Scheduler**: genera `N` eventos externos con timestamps estrictamente crecientes y los asigna
  aleatoriamente a los workers. También mantiene un reloj virtual y deja trazas de los envíos.
- **Workers**: mantienen su propio *Local Virtual Time* (LVT), un historial ordenado de eventos
  externos, una pila de checkpoints y un generador de eventos internos que provocan saltos en el LVT.
  Cuando un worker recibe un evento externo con timestamp menor a su LVT actual, detecta el
  straggler, restaura el checkpoint más cercano, y re-procesa los eventos externos a partir de ese
  punto.
- **Logging estructurado**: cada acción relevante (envíos, recepciones, checkpoints, eventos
  internos, rollbacks) queda registrada como una línea JSON en un archivo `execution.log` (o en la
  ruta que se configure).
- **Visualización offline** (`visualize.py`): lee el log JSONL y genera un diagrama espacio-tiempo
  donde se aprecian los eventos por entidad y los rollbacks.

## Requisitos

- Go 1.21+ para compilar `main.go`.
- Python 3.9+ y `matplotlib` para el script de visualización.

## Ejecución de la simulación

Compilar o ejecutar directamente con `go run`:

```bash
go run . -workers 3 -events 60 -log execution.jsonl
```

Parámetros principales:

- `-workers`: número de workers simultáneos.
- `-events`: cantidad total de eventos externos creados por el scheduler.
- `-log`: ruta del archivo JSON Lines que almacenará la traza.
- `-internal-min/-internal-max`: cantidad de eventos internos generados por cada evento externo.
- `-jump-min/-jump-max`: cuánto avanza el LVT cuando se ejecuta un evento interno.
- `-channel-buffer`: tamaño de los canales Scheduler→Worker.
- `-seed`: semilla para reproducir ejecuciones.
- `-speedup`: si se pasa este flag, el programa ejecuta automáticamente la simulación con 1, 2, 4 y
  8 workers, guarda un log por corrida y muestra la tabla de *speedup* relativo al caso de 1 worker.

Al finalizar, el programa muestra estadísticas por worker: eventos externos procesados, eventos
internos generados, cantidad de rollbacks, checkpoints creados y LVT final.

## Formato de log

Cada línea del archivo es un objeto JSON con los campos más relevantes:

- `entity`: `scheduler` o `worker-X`.
- `event`: tipo de evento (`external_dispatched`, `external_received`, `checkpoint_created`,
  `internal_processed`, `rollback_start`, etc.).
- `sim_time`: LVT de la entidad al momento del log.
- `event_id`, `target_worker`: identificadores útiles para correlacionar envíos y recepciones.
- `rollback_from` / `rollback_to`: tiempos utilizados al representar la operación de rollback.
- `details`: mapa con información específica según el evento (p.ej. saltos de LVT en eventos
  internos, si un evento se procesó durante un replay, etc.).

## Visualización de la ejecución

El script `visualize.py` genera un diagrama espacio-tiempo a partir del log JSONL:

```bash
python3 visualize.py execution.jsonl -o timeline.png --show
```

- El eje X representa el tiempo virtual (LVT) y el eje Y cada entidad (scheduler y workers).
- Los colores y marcadores distinguen los diferentes tipos de eventos.
- Los rollbacks aparecen como flechas que apuntan desde el tiempo original hacia el tiempo al que se
  regresa.

## Análisis de escalabilidad

Para medir el *speedup* automáticamente, ejecutar:

```bash
go run . -events 120 -speedup
```

La aplicación correrá 4 simulaciones (1, 2, 4 y 8 workers), generará los logs
`speedup_w1.log`, `speedup_w2.log`, etc., y mostrará los tiempos medidos junto con el *speedup*
calculado respecto al caso de un worker.

## Desarrollo futuro

- Ajustar la generación de eventos internos/external para simular topologías reales.
- Añadir opciones de persistencia de checkpoints/rollbacks a disco si se requieren ejecuciones más
  largas.
- Integrar métricas adicionales (latencias promedio, varianza de LVT entre workers, etc.) para un
  análisis más detallado.
