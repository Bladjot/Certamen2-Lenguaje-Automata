# Simulador Distribuido Time Warp

Este programa implementa un simulador completo del algoritmo **Time Warp** para simulación distribuida de eventos discretos en Go. El simulador incluye manejo de eventos, rollback automático, persistencia de estados mediante checkpoints, logging thread-safe, visualización de diagramas espacio-tiempo y sistema de benchmarking para análisis de rendimiento.

## 🏗️ Arquitectura del Sistema

### Componentes Principales

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│    Scheduler    │────│     Workers      │────│   Checkpoints   │
│   (Coordinator) │    │ (Time Warp Proc)│    │ (State Persist) │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│     Events      │    │     Logger       │    │  Visualization  │
│  (Temporal Ord) │    │ (Thread-Safe)    │    │   (CSV/PNG)     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

### Módulos del Sistema

- **`event/`**: Sistema de eventos con ordenamiento temporal
- **`worker/`**: Implementación de workers Time Warp con rollback
- **`scheduler/`**: Coordinador para distribución de eventos
- **`checkpoint/`**: Persistencia de estados usando encoding/gob
- **`log/`**: Sistema de logging thread-safe con formato JSON
- **`visualization/`**: Procesamiento de logs y generación de CSV

## 🚀 Instalación y Uso

### Prerrequisitos

- Go 1.25.3 o superior
- Python 3.x con matplotlib (para visualización)

### Instalación

```bash
# Clonar el repositorio
git clone <repository-url>
cd Certamen2-Lenguaje-Automata

# Verificar instalación de Go
go version

# Instalar dependencias de Python para visualización
pip install matplotlib pandas
```

### Ejecución del Simulador

#### Modo Normal (Simulación Completa)

```bash
# Ejecutar simulación con configuración por defecto
go run .

# El simulador ejecutará:
# - 4 workers concurrentes
# - 5 iteraciones de simulación
# - 3 eventos por iteración
# - Generación automática de logs y visualizaciones
```

#### Modo Benchmark (Análisis de Rendimiento)

```bash
# Ejecutar suite completa de benchmarks
go run . benchmark

# Se ejecutarán configuraciones de:
# - 1, 2, 4, 8 workers con 15 eventos
# - 1, 4 workers con 30 eventos
# - Cálculo automático de speedup y eficiencia
```

### Generación de Visualizaciones

Después de ejecutar el simulador, generar diagramas espacio-tiempo:

```bash
# Procesar logs y generar diagramas PNG
python visualize.py

# Archivos generados:
# - execution_timeline.png (diagrama principal)
# - execution_timeline_detailed.png (vista detallada)
# - execution_timeline_stats.txt (estadísticas)
```

## 📊 Archivos de Salida

### Logs y Datos

- **`logs.json`**: Logs de ejecución en formato JSON con timestamps
- **`execution.csv`**: Datos procesados para análisis temporal
- **`thread_summary.csv`**: Resumen estadístico por worker
- **`speedup.csv`**: Resultados de benchmarks con métricas de rendimiento

### Visualizaciones

- **`execution_timeline.png`**: Diagrama espacio-tiempo principal
- **`execution_timeline_detailed.png`**: Vista detallada con todos los eventos
- **`execution_timeline_stats.txt`**: Estadísticas textuales del análisis

### Checkpoints

- **`checkpoints/`**: Directorio con estados persistidos de workers
- **`benchmark_*.json`**: Logs individuales de cada configuración de benchmark

## 🔧 Algoritmo Time Warp

### Conceptos Fundamentales

#### 1. **Local Virtual Time (LVT)**
Cada worker mantiene su tiempo local virtual que avanza con el procesamiento de eventos.

#### 2. **Straggler Detection**
Detección automática de eventos que llegan "tarde" (timestamp menor al LVT actual).

#### 3. **Rollback Mechanism**
```
Si evento.timestamp < worker.LVT:
    1. Detectar straggler
    2. Buscar checkpoint más cercano
    3. Restaurar estado desde checkpoint  
    4. Recalcular eventos desde el punto de rollback
    5. Actualizar LVT y continuar simulación
```

#### 4. **Checkpoint Strategy**
- Checkpoint automático cada 5 eventos procesados
- Persistencia usando `encoding/gob` para serialización eficiente
- Limpieza automática de checkpoints antiguos

### Flujo de Ejecución

```
1. Inicialización
   ├── Crear workers (goroutines)
   ├── Inicializar scheduler
   └── Configurar sistema de logging

2. Simulación
   ├── Scheduler genera eventos EXTERNAL
   ├── Workers procesan eventos concurrentemente
   ├── Detección automática de stragglers
   ├── Rollback y recuperación de estado
   └── Logging de todas las acciones

3. Finalización
   ├── Sincronización de workers (WaitGroup)
   ├── Procesamiento de logs a CSV
   ├── Generación de visualizaciones
   └── Análisis de rendimiento (opcional)
```

## 📈 Sistema de Benchmarking

### Métricas Calculadas

#### **Speedup**
```
Speedup = T_secuencial / T_paralelo
```
Donde:
- `T_secuencial`: Tiempo de ejecución con 1 worker
- `T_paralelo`: Tiempo de ejecución con N workers

#### **Eficiencia**
```
Eficiencia = Speedup / N_workers
```
Valor ideal = 1.0 (speedup lineal perfecto)

### Configuraciones de Benchmark

| Workers | Eventos | Propósito |
|---------|---------|-----------|
| 1       | 15      | Baseline secuencial |
| 2       | 15      | Paralelismo básico |
| 4       | 15      | Configuración estándar |
| 8       | 15      | Máximo paralelismo |
| 1       | 30      | Carga alta secuencial |
| 4       | 30      | Carga alta paralela |

## 🎨 Visualización de Resultados

### Diagramas Espacio-Tiempo

Los diagramas generados muestran:

- **Eje X**: Tiempo de simulación (LVT)
- **Eje Y**: Workers/Threads (Scheduler + Workers 0-3)
- **Colores**: Diferentes tipos de acciones
  - 🟢 Verde: Procesamiento normal de eventos
  - 🔴 Rojo: Detección de stragglers
  - 🟡 Amarillo: Operaciones de rollback
  - 🔵 Azul: Creación de checkpoints
  - 🟣 Morado: Generación de eventos externos

### Interpretación de Resultados

1. **Líneas Paralelas**: Indican procesamiento concurrente eficiente
2. **Rollbacks Frecuentes**: Pueden indicar alta contención o eventos mal distribuidos
3. **Checkpoints Regulares**: Muestran estrategia de persistencia funcionando
4. **Gaps Temporales**: Períodos de inactividad o sincronización

## 🔍 Estructura de Código

### Archivos Principales

```
main.go                 # Orquestación principal y benchmarks
├── event/
│   └── event.go        # Definición de eventos y ordenamiento temporal
├── worker/
│   └── worker.go       # Implementación Time Warp con rollback
├── scheduler/
│   └── scheduler.go    # Coordinador de simulación
├── checkpoint/
│   └── checkpoint.go   # Sistema de persistencia de estados
├── log/
│   └── logger.go       # Logging thread-safe con JSON
└── visualization/
    └── parser.go       # Procesamiento de logs a CSV
```

### Configuración del Sistema

```go
const (
    NUM_WORKERS           = 4                    // Número de workers concurrentes
    SIMULATION_ITERATIONS = 5                    // Iteraciones de simulación
    EVENTS_PER_ITERATION  = 3                    // Eventos por iteración
    SLEEP_DURATION        = 2 * time.Second     // Pausa entre iteraciones
)
```

## 🚦 Casos de Uso

### 1. Investigación Académica
- Estudio del comportamiento del algoritmo Time Warp
- Análisis de eficiencia de rollback en diferentes cargas
- Comparación de estrategias de checkpoint

### 2. Análisis de Rendimiento
- Medición de speedup en sistemas multi-core
- Evaluación de overhead de sincronización
- Optimización de parámetros de simulación

### 3. Visualización de Algoritmos
- Comprensión visual del ordenamiento causal
- Análisis de patrones de rollback
- Debuging de simulaciones distribuidas

## 🔧 Personalización

### Modificar Configuración

Para cambiar el comportamiento del simulador, editar las constantes en `main.go`:

```go
const (
    NUM_WORKERS           = 8    // Aumentar paralelismo
    SIMULATION_ITERATIONS = 10   // Más iteraciones
    EVENTS_PER_ITERATION  = 5    // Más eventos por ciclo
)
```

### Añadir Nuevos Tipos de Eventos

En `event/event.go`, extender el enum `EventType`:

```go
type EventType int

const (
    EXTERNAL EventType = iota
    INTERNAL
    CUSTOM_EVENT  // Nuevo tipo
)
```

### Personalizar Estrategia de Checkpoint

En `worker/worker.go`, modificar la frecuencia:

```go
const CHECKPOINT_FREQUENCY = 10  // Checkpoint cada 10 eventos
```

## 📝 Troubleshooting

### Problemas Comunes

1. **Error "too many open files"**
   ```bash
   ulimit -n 2048  # Aumentar límite de archivos abiertos
   ```

2. **Visualización no se genera**
   ```bash
   pip install --upgrade matplotlib pandas
   python visualize.py
   ```

3. **Benchmarks muy lentos**
   - Reducir `SIMULATION_ITERATIONS` y `EVENTS_PER_ITERATION`
   - Ajustar `SLEEP_DURATION` a un valor menor

4. **Workers no sincronizan correctamente**
   - Verificar que `sync.WaitGroup` esté configurado correctamente
   - Revisar logs en `logs.json` para identificar deadlocks

## 📚 Referencias

- Jefferson, D. R. (1985). "Virtual Time". ACM Transactions on Programming Languages and Systems.
- Fujimoto, R. M. (2000). "Parallel and Distribution Simulation Systems". Wiley.
- Documentación oficial de Go: https://golang.org/doc/

---

## 🏆 Características Destacadas

✅ **Algoritmo Time Warp completo** con rollback automático  
✅ **Sistema de checkpoints** con persistencia usando encoding/gob  
✅ **Logging thread-safe** con formato JSON estructurado  
✅ **Visualización avanzada** con diagramas espacio-tiempo  
✅ **Benchmarking integrado** con cálculo de speedup y eficiencia  
✅ **Arquitectura modular** y fácilmente extensible  
✅ **Documentación completa** con ejemplos de uso  

**Desarrollado en Go 1.25.3** | **Compatible con sistemas Unix/Linux/macOS**