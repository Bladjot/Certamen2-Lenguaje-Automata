package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"certamen2/checkpoint"
	"certamen2/event"
	customlog "certamen2/log"
	"certamen2/scheduler"
	"certamen2/visualization"
	"certamen2/worker"
)

const (
	// Configuración de la simulación
	NUM_WORKERS           = 4
	SIMULATION_ITERATIONS = 5
	EVENTS_PER_ITERATION  = 3
	SLEEP_DURATION        = 2 * time.Second

	// Rutas de archivos
	LOGS_PATH        = "logs.json"
	CSV_PATH         = "execution.csv"
	SUMMARY_PATH     = "thread_summary.csv"
	CHECKPOINTS_PATH = "checkpoints"
)

func main() {
	// Verificar si se solicita benchmark
	if len(os.Args) > 1 && os.Args[1] == "benchmark" {
		fmt.Println("=== MODO BENCHMARK ===")
		if err := runBenchmarkSuite(); err != nil {
			log.Fatalf("Error ejecutando benchmark: %v", err)
		}
		return
	}

	fmt.Println("=== SIMULADOR DISTRIBUIDO TIME WARP ===")
	fmt.Printf("Configuración: %d workers, %d iteraciones, %d eventos/iteración\n",
		NUM_WORKERS, SIMULATION_ITERATIONS, EVENTS_PER_ITERATION)

	// Inicializar sistema de logging
	fmt.Println("\n1. Inicializando sistema de logging...")
	if err := customlog.InitLogger(); err != nil {
		log.Fatalf("Error inicializando logger: %v", err)
	}

	// Log de inicio del sistema
	customlog.LogSchedulerEvent("INIT", 0, "Start", 0, "Simulador iniciado")

	// Crear workers
	fmt.Printf("\n2. Creando %d workers...\n", NUM_WORKERS)
	workers := make([]*worker.Worker, NUM_WORKERS)
	var wg sync.WaitGroup

	for i := 0; i < NUM_WORKERS; i++ {
		workers[i] = worker.NewWorker(i)
		fmt.Printf("   Worker %d creado\n", i)

		// Log de creación de worker
		customlog.LogWorkerEvent(i, "INIT", 0, "Start", 0,
			fmt.Sprintf("Worker %d inicializado", i))
	}

	// Crear scheduler
	fmt.Println("\n3. Creando scheduler...")
	sched := scheduler.NewScheduler(workers)
	fmt.Println("   Scheduler creado con", len(workers), "workers")

	// Log de creación de scheduler
	customlog.LogSchedulerEvent("INIT", 0, "Start", 0, "Scheduler inicializado")

	// Configurar WaitGroup para esperar que todos los workers terminen
	wg.Add(NUM_WORKERS)

	// Lanzar workers como goroutines
	fmt.Println("\n4. Lanzando workers como goroutines...")
	for i, w := range workers {
		go func(workerID int, worker *worker.Worker) {
			defer wg.Done()
			fmt.Printf("   Goroutine del Worker %d iniciada\n", workerID)

			// Log de inicio de goroutine
			customlog.LogWorkerEvent(workerID, "GOROUTINE", 0, "Start", 0,
				fmt.Sprintf("Goroutine Worker %d iniciada", workerID))

			worker.Run()

			// Log de fin de goroutine
			customlog.LogWorkerEvent(workerID, "GOROUTINE", 0, "End", 0,
				fmt.Sprintf("Goroutine Worker %d terminada", workerID))

			fmt.Printf("   Goroutine del Worker %d terminada\n", workerID)
		}(i, w)
	}

	// Esperar un momento para que las goroutines se inicialicen
	time.Sleep(500 * time.Millisecond)

	// Ejecutar simulación principal
	fmt.Println("\n5. Ejecutando simulación principal...")
	customlog.LogSchedulerEvent("SIMULATION", 0, "Start", 0, "Iniciando simulación principal")

	// Ejecutar scheduler (esto bloqueará hasta que termine)
	sched.Run(SIMULATION_ITERATIONS, EVENTS_PER_ITERATION, SLEEP_DURATION)

	fmt.Println("\n6. Esperando que todos los workers terminen...")
	customlog.LogSchedulerEvent("SIMULATION", 0, "WaitingWorkers", 0, "Esperando workers")

	// Esperar que todas las goroutines terminen
	wg.Wait()

	fmt.Println("\n7. Todos los workers han terminado")
	customlog.LogSchedulerEvent("SIMULATION", 0, "End", 0, "Simulación completada")

	// Mostrar estadísticas finales
	fmt.Println("\n=== ESTADÍSTICAS FINALES ===")
	fmt.Printf("Scheduler: %s\n", sched.GetStatus())

	for _, w := range workers {
		fmt.Printf("Worker %d: %s\n", w.ID, w.GetStatus())
	}

	// Cerrar sistema de logging
	fmt.Println("\n8. Cerrando sistema de logging...")
	customlog.LogSchedulerEvent("SHUTDOWN", 0, "End", 0, "Sistema cerrando")
	if err := customlog.CloseLogger(); err != nil {
		log.Printf("Error cerrando logger: %v", err)
	}

	// Procesar logs para visualización
	fmt.Println("\n9. Procesando logs para visualización...")
	if err := processLogsForVisualization(); err != nil {
		log.Printf("Error procesando logs: %v", err)
	} else {
		fmt.Println("   Logs procesados exitosamente")
		fmt.Printf("   - Archivo CSV generado: %s\n", CSV_PATH)
		fmt.Printf("   - Resumen generado: %s\n", SUMMARY_PATH)
		fmt.Println("   - Para generar diagrama, ejecuta: python3 visualize.py")
	}

	// Mostrar información de checkpoints
	fmt.Println("\n10. Información de checkpoints...")
	showCheckpointInfo()

	fmt.Println("\n=== SIMULACIÓN COMPLETADA ===")
	fmt.Println("Archivos generados:")
	fmt.Printf("   - Logs: %s\n", LOGS_PATH)
	fmt.Printf("   - CSV: %s\n", CSV_PATH)
	fmt.Printf("   - Resumen: %s\n", SUMMARY_PATH)
	fmt.Printf("   - Checkpoints: directorio %s/\n", CHECKPOINTS_PATH)
	fmt.Println("\nPara visualizar:")
	fmt.Println("   python3 visualize.py")
}

// processLogsForVisualization procesa los logs JSON y genera archivos CSV
func processLogsForVisualization() error {
	return visualization.ProcessLogs(LOGS_PATH, CSV_PATH, SUMMARY_PATH)
}

// showCheckpointInfo muestra información sobre checkpoints generados
func showCheckpointInfo() {
	fmt.Printf("   Directorio de checkpoints: %s\n", CHECKPOINTS_PATH)

	// Intentar mostrar checkpoints para cada worker
	for i := 0; i < NUM_WORKERS; i++ {
		timestamps, err := checkpoint.ListCheckpoints(i, CHECKPOINTS_PATH)
		if err != nil {
			fmt.Printf("   Worker %d: Sin checkpoints\n", i)
		} else {
			fmt.Printf("   Worker %d: %d checkpoints encontrados\n", i, len(timestamps))
		}
	}
}

// demoCheckpointOperations demuestra operaciones de checkpoint (opcional)
func demoCheckpointOperations() {
	fmt.Println("\n=== DEMO CHECKPOINT ===")

	// Crear un snapshot de ejemplo
	variables := map[string]int{
		"counter":   42,
		"state":     1,
		"processed": 10,
	}

	snapshot := checkpoint.NewStateSnapshot(0, 100, variables, 100, 10)

	// Guardar checkpoint
	if err := checkpoint.SaveCheckpoint(0, snapshot, CHECKPOINTS_PATH); err != nil {
		fmt.Printf("Error guardando checkpoint: %v\n", err)
		return
	}

	fmt.Println("Checkpoint de demo guardado")

	// Cargar checkpoint
	loaded, err := checkpoint.LoadCheckpoint(0, 100, CHECKPOINTS_PATH)
	if err != nil {
		fmt.Printf("Error cargando checkpoint: %v\n", err)
		return
	}

	fmt.Printf("Checkpoint cargado: %s\n", loaded.String())
}

// demoEventOperations demuestra operaciones con eventos (opcional)
func demoEventOperations() {
	fmt.Println("\n=== DEMO EVENTOS ===")

	// Crear eventos de ejemplo
	e1 := event.NewEvent(1, event.EXTERNAL, 10, 0)
	e2 := event.NewEvent(2, event.INTERNAL, 15, 1)
	e3 := event.NewEvent(3, event.EXTERNAL, 12, 0)

	fmt.Printf("Evento 1: %s\n", e1.String())
	fmt.Printf("Evento 2: %s\n", e2.String())
	fmt.Printf("Evento 3: %s\n", e3.String())

	// Comparar eventos
	fmt.Printf("Comparar e1 vs e2: %d\n", event.Compare(e1, e2))
	fmt.Printf("Comparar e1 vs e3: %d\n", event.Compare(e1, e3))
	fmt.Printf("Comparar e3 vs e2: %d\n", event.Compare(e3, e2))
}

// benchmarkSimulation ejecuta la simulación con diferentes configuraciones y mide rendimiento
func benchmarkSimulation(numWorkers int, numEvents int) (time.Duration, float64, error) {
	fmt.Printf("\n=== BENCHMARK: %d workers, %d eventos ===\n", numWorkers, numEvents)

	// Configurar logging silencioso para benchmark
	originalLogPath := customlog.GetLogPath()
	benchmarkLogPath := fmt.Sprintf("benchmark_%dw_%de.json", numWorkers, numEvents)
	customlog.SetLogPath(benchmarkLogPath)

	// Inicializar logger para benchmark
	if err := customlog.InitLogger(); err != nil {
		return 0, 0, fmt.Errorf("error inicializando logger: %v", err)
	}
	defer customlog.CloseLogger()

	// Crear workers
	workers := make([]*worker.Worker, numWorkers)
	for i := 0; i < numWorkers; i++ {
		workers[i] = worker.NewWorker(i)
	}

	// Crear scheduler
	sched := scheduler.NewScheduler(workers)

	// Configurar WaitGroup
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	// Iniciar medición de tiempo
	startTime := time.Now()

	// Lanzar workers
	for i, w := range workers {
		go func(workerID int, worker *worker.Worker) {
			defer wg.Done()
			worker.Run()
		}(i, w)
	}

	// Calcular iteraciones basado en número de eventos deseados
	eventsPerIteration := 3
	iterations := (numEvents + eventsPerIteration - 1) / eventsPerIteration // ceiling division

	// Ejecutar simulación sin delay para benchmark
	sched.Run(iterations, eventsPerIteration, 100*time.Millisecond)

	// Esperar workers
	wg.Wait()

	// Finalizar medición
	duration := time.Since(startTime)

	// Restaurar configuración original
	customlog.SetLogPath(originalLogPath)

	fmt.Printf("Benchmark completado en %v\n", duration)
	return duration, 0, nil // Speedup se calculará externamente
}

// runBenchmarkSuite ejecuta una suite completa de benchmarks y guarda resultados
func runBenchmarkSuite() error {
	fmt.Println("\n=== SUITE DE BENCHMARKS ===")

	// Configuraciones de benchmark
	benchmarkConfigs := []struct {
		workers int
		events  int
	}{
		{1, 15}, // Secuencial
		{2, 15}, // 2 workers
		{4, 15}, // 4 workers
		{8, 15}, // 8 workers
		{1, 30}, // Más eventos secuencial
		{4, 30}, // Más eventos paralelo
	}

	// Ejecutar benchmark secuencial como referencia
	fmt.Println("Ejecutando benchmark secuencial de referencia...")
	baselineDuration, _, err := benchmarkSimulation(1, 15)
	if err != nil {
		return fmt.Errorf("error en benchmark baseline: %v", err)
	}

	// Crear archivo CSV para resultados
	csvFile, err := os.Create("speedup.csv")
	if err != nil {
		return fmt.Errorf("error creando speedup.csv: %v", err)
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	// Escribir header CSV
	header := []string{"Workers", "Events", "Duration_ms", "Speedup", "Efficiency"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("error escribiendo header CSV: %v", err)
	}

	// Ejecutar benchmarks y calcular speedups
	for _, config := range benchmarkConfigs {
		duration, _, err := benchmarkSimulation(config.workers, config.events)
		if err != nil {
			fmt.Printf("Error en benchmark %d workers, %d eventos: %v\n",
				config.workers, config.events, err)
			continue
		}

		// Calcular speedup relativo al baseline
		var speedup float64
		var efficiency float64

		if config.events == 15 { // Comparar con baseline del mismo número de eventos
			speedup = float64(baselineDuration.Nanoseconds()) / float64(duration.Nanoseconds())
			efficiency = speedup / float64(config.workers)
		} else {
			speedup = 1.0 // Para diferentes números de eventos, no calcular speedup
			efficiency = 1.0
		}

		// Escribir registro CSV
		record := []string{
			strconv.Itoa(config.workers),
			strconv.Itoa(config.events),
			strconv.FormatFloat(float64(duration.Nanoseconds())/1e6, 'f', 2, 64), // ms
			strconv.FormatFloat(speedup, 'f', 3, 64),
			strconv.FormatFloat(efficiency, 'f', 3, 64),
		}

		if err := writer.Write(record); err != nil {
			fmt.Printf("Error escribiendo registro CSV: %v\n", err)
		}

		fmt.Printf("Workers: %d, Eventos: %d, Duración: %v, Speedup: %.3f, Eficiencia: %.3f\n",
			config.workers, config.events, duration, speedup, efficiency)
	}

	fmt.Printf("\nResultados guardados en speedup.csv\n")
	return nil
}

// init se ejecuta antes de main() para configuraciones iniciales
func init() {
	// Configurar logging estándar
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Mensaje de inicio
	fmt.Println("Inicializando simulador distribuido...")
}
