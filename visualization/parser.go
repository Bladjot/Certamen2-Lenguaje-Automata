package visualization

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
)

// LogEntry representa una entrada de log (debe coincidir con log/logger.go)
type LogEntry struct {
	ThreadType string `json:"thread_type"`
	ThreadID   int    `json:"thread_id"`
	EventType  string `json:"event_type"`
	Timestamp  int    `json:"timestamp"`
	Action     string `json:"action"`
	LVT        int    `json:"lvt"`
	Message    string `json:"message"`
	CreatedAt  string `json:"created_at"`
}

// TimelinePoint representa un punto en la línea de tiempo para visualización
type TimelinePoint struct {
	ThreadType string
	ThreadID   int
	Timestamp  int
	Action     string
	EventType  string
	LVT        int
	Message    string
	CreatedAt  string
}

// ThreadData agrupa datos por thread para análisis
type ThreadData struct {
	ThreadType string
	ThreadID   int
	Points     []TimelinePoint
}

// ExecutionData contiene todos los datos de ejecución estructurados
type ExecutionData struct {
	Threads   map[string]*ThreadData // Key: "ThreadType_ThreadID"
	AllPoints []TimelinePoint        // Todos los puntos ordenados por timestamp
}

// ParseLogs lee logs.json línea por línea y construye estructura de datos
func ParseLogs(logFilePath string) (*ExecutionData, error) {
	log.Printf("Parser: Leyendo logs desde %s", logFilePath)

	// Verificar que el archivo existe
	if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("archivo de logs no encontrado: %s", logFilePath)
	}

	// Abrir archivo de logs
	file, err := os.Open(logFilePath)
	if err != nil {
		return nil, fmt.Errorf("error abriendo archivo de logs: %v", err)
	}
	defer file.Close()

	// Inicializar estructura de datos
	execData := &ExecutionData{
		Threads:   make(map[string]*ThreadData),
		AllPoints: make([]TimelinePoint, 0),
	}

	// Leer línea por línea
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())

		// Ignorar líneas vacías o comentarios
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// Parsear entrada JSON
		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			log.Printf("Parser: Error parseando línea %d: %v - Línea: %s", lineNumber, err, line)
			continue // Continuar con la siguiente línea
		}

		// Crear punto de timeline
		point := TimelinePoint{
			ThreadType: entry.ThreadType,
			ThreadID:   entry.ThreadID,
			Timestamp:  entry.Timestamp,
			Action:     entry.Action,
			EventType:  entry.EventType,
			LVT:        entry.LVT,
			Message:    entry.Message,
			CreatedAt:  entry.CreatedAt,
		}

		// Agregar a la lista de todos los puntos
		execData.AllPoints = append(execData.AllPoints, point)

		// Agrupar por thread
		threadKey := fmt.Sprintf("%s_%d", entry.ThreadType, entry.ThreadID)
		if execData.Threads[threadKey] == nil {
			execData.Threads[threadKey] = &ThreadData{
				ThreadType: entry.ThreadType,
				ThreadID:   entry.ThreadID,
				Points:     make([]TimelinePoint, 0),
			}
		}

		execData.Threads[threadKey].Points = append(execData.Threads[threadKey].Points, point)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error leyendo archivo: %v", err)
	}

	// Ordenar puntos por timestamp
	sort.Slice(execData.AllPoints, func(i, j int) bool {
		return execData.AllPoints[i].Timestamp < execData.AllPoints[j].Timestamp
	})

	// Ordenar puntos dentro de cada thread
	for _, threadData := range execData.Threads {
		sort.Slice(threadData.Points, func(i, j int) bool {
			return threadData.Points[i].Timestamp < threadData.Points[j].Timestamp
		})
	}

	log.Printf("Parser: Procesadas %d entradas de log, %d threads identificados",
		len(execData.AllPoints), len(execData.Threads))

	return execData, nil
}

// GenerateCSV guarda los resultados en un archivo CSV intermedio
func GenerateCSV(execData *ExecutionData, outputPath string) error {
	log.Printf("Parser: Generando CSV en %s", outputPath)

	// Crear directorio si no existe
	if strings.Contains(outputPath, "/") {
		dirPath := outputPath[:strings.LastIndex(outputPath, "/")]
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("error creando directorio: %v", err)
		}
	}

	// Crear archivo CSV
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error creando archivo CSV: %v", err)
	}
	defer file.Close()

	// Crear writer CSV
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Escribir header
	header := []string{
		"ThreadType",
		"ThreadID",
		"Timestamp",
		"Action",
		"EventType",
		"LVT",
		"Message",
		"CreatedAt",
	}

	if err := writer.Write(header); err != nil {
		return fmt.Errorf("error escribiendo header CSV: %v", err)
	}

	// Escribir datos (todos los puntos ordenados por timestamp)
	for _, point := range execData.AllPoints {
		record := []string{
			point.ThreadType,
			strconv.Itoa(point.ThreadID),
			strconv.Itoa(point.Timestamp),
			point.Action,
			point.EventType,
			strconv.Itoa(point.LVT),
			point.Message,
			point.CreatedAt,
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("error escribiendo registro CSV: %v", err)
		}
	}

	log.Printf("Parser: CSV generado exitosamente con %d registros", len(execData.AllPoints))
	return nil
}

// GenerateThreadSummary genera un resumen por thread
func GenerateThreadSummary(execData *ExecutionData, outputPath string) error {
	log.Printf("Parser: Generando resumen por thread en %s", outputPath)

	// Crear archivo de resumen
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error creando archivo de resumen: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Header para resumen
	header := []string{
		"ThreadType",
		"ThreadID",
		"TotalEvents",
		"FirstTimestamp",
		"LastTimestamp",
		"TimeSpan",
		"SendActions",
		"ReceiveActions",
		"CheckpointActions",
		"RollbackActions",
	}

	if err := writer.Write(header); err != nil {
		return fmt.Errorf("error escribiendo header de resumen: %v", err)
	}

	// Procesar cada thread
	for _, threadData := range execData.Threads {
		if len(threadData.Points) == 0 {
			continue
		}

		// Calcular estadísticas
		firstTimestamp := threadData.Points[0].Timestamp
		lastTimestamp := threadData.Points[len(threadData.Points)-1].Timestamp
		timeSpan := lastTimestamp - firstTimestamp

		// Contar acciones
		sendCount := 0
		receiveCount := 0
		checkpointCount := 0
		rollbackCount := 0

		for _, point := range threadData.Points {
			switch point.Action {
			case "Send":
				sendCount++
			case "Receive":
				receiveCount++
			case "Checkpoint":
				checkpointCount++
			case "Rollback":
				rollbackCount++
			}
		}

		// Escribir registro de resumen
		record := []string{
			threadData.ThreadType,
			strconv.Itoa(threadData.ThreadID),
			strconv.Itoa(len(threadData.Points)),
			strconv.Itoa(firstTimestamp),
			strconv.Itoa(lastTimestamp),
			strconv.Itoa(timeSpan),
			strconv.Itoa(sendCount),
			strconv.Itoa(receiveCount),
			strconv.Itoa(checkpointCount),
			strconv.Itoa(rollbackCount),
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("error escribiendo registro de resumen: %v", err)
		}
	}

	log.Printf("Parser: Resumen por thread generado exitosamente")
	return nil
}

// ProcessLogs función principal que ejecuta todo el pipeline de procesamiento
func ProcessLogs(logFilePath, csvOutputPath, summaryOutputPath string) error {
	log.Printf("Parser: Iniciando procesamiento de logs")

	// Parsear logs
	execData, err := ParseLogs(logFilePath)
	if err != nil {
		return fmt.Errorf("error parseando logs: %v", err)
	}

	// Generar CSV principal
	if err := GenerateCSV(execData, csvOutputPath); err != nil {
		return fmt.Errorf("error generando CSV: %v", err)
	}

	// Generar resumen por thread
	if err := GenerateThreadSummary(execData, summaryOutputPath); err != nil {
		return fmt.Errorf("error generando resumen: %v", err)
	}

	log.Printf("Parser: Procesamiento completado exitosamente")
	return nil
}

// GetExecutionStatistics calcula estadísticas generales de la ejecución
func GetExecutionStatistics(execData *ExecutionData) map[string]interface{} {
	stats := make(map[string]interface{})

	if len(execData.AllPoints) == 0 {
		return stats
	}

	// Estadísticas generales
	stats["total_events"] = len(execData.AllPoints)
	stats["total_threads"] = len(execData.Threads)
	stats["first_timestamp"] = execData.AllPoints[0].Timestamp
	stats["last_timestamp"] = execData.AllPoints[len(execData.AllPoints)-1].Timestamp
	stats["execution_span"] = execData.AllPoints[len(execData.AllPoints)-1].Timestamp - execData.AllPoints[0].Timestamp

	// Contar por tipo de thread
	schedulerCount := 0
	workerCount := 0
	for _, threadData := range execData.Threads {
		if threadData.ThreadType == "Scheduler" {
			schedulerCount++
		} else if threadData.ThreadType == "Worker" {
			workerCount++
		}
	}

	stats["scheduler_threads"] = schedulerCount
	stats["worker_threads"] = workerCount

	// Contar por tipo de acción
	actionCounts := make(map[string]int)
	for _, point := range execData.AllPoints {
		actionCounts[point.Action]++
	}
	stats["action_counts"] = actionCounts

	return stats
}

// PrintStatistics imprime estadísticas en formato legible
func PrintStatistics(stats map[string]interface{}) {
	fmt.Println("=== ESTADÍSTICAS DE EJECUCIÓN ===")
	fmt.Printf("Total de eventos: %v\n", stats["total_events"])
	fmt.Printf("Total de threads: %v\n", stats["total_threads"])
	fmt.Printf("Threads Scheduler: %v\n", stats["scheduler_threads"])
	fmt.Printf("Threads Worker: %v\n", stats["worker_threads"])
	fmt.Printf("Timestamp inicial: %v\n", stats["first_timestamp"])
	fmt.Printf("Timestamp final: %v\n", stats["last_timestamp"])
	fmt.Printf("Duración total: %v\n", stats["execution_span"])

	if actionCounts, ok := stats["action_counts"].(map[string]int); ok {
		fmt.Println("\nConteo por acción:")
		for action, count := range actionCounts {
			fmt.Printf("  %s: %d\n", action, count)
		}
	}
}
