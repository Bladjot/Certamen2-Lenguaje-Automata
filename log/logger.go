package log

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// LogEntry representa una entrada de log del simulador
type LogEntry struct {
	ThreadType string `json:"thread_type"` // "Scheduler" o "Worker"
	ThreadID   int    `json:"thread_id"`   // ID del thread
	EventType  string `json:"event_type"`  // Tipo de evento relacionado
	Timestamp  int    `json:"timestamp"`   // Timestamp del evento
	Action     string `json:"action"`      // "Send", "Receive", "Checkpoint", "Rollback"
	LVT        int    `json:"lvt"`         // Local Virtual Time
	Message    string `json:"message"`     // Mensaje descriptivo adicional
	CreatedAt  string `json:"created_at"`  // Timestamp real de cuando se creó el log
}

// Logger global para el simulador
var (
	logMutex sync.Mutex               // Mutex para escritura segura
	logFile  *os.File                 // Archivo de log
	logPath  string     = "logs.json" // Ruta del archivo de log
)

// InitLogger inicializa el sistema de logging y limpia logs previos
func InitLogger() error {
	logMutex.Lock()
	defer logMutex.Unlock()

	// Cerrar archivo anterior si existe
	if logFile != nil {
		logFile.Close()
	}

	// Crear/truncar el archivo de log
	file, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("error creando archivo de log %s: %v", logPath, err)
	}

	logFile = file

	// Escribir header informativo
	header := LogEntry{
		ThreadType: "System",
		ThreadID:   0,
		EventType:  "INIT",
		Timestamp:  0,
		Action:     "Start",
		LVT:        0,
		Message:    "Simulador iniciado - Logging habilitado",
		CreatedAt:  time.Now().Format("2006-01-02 15:04:05.000"),
	}

	// Escribir header como primera línea
	headerJSON, _ := json.Marshal(header)
	logFile.Write(headerJSON)
	logFile.Write([]byte("\n"))
	logFile.Sync()

	fmt.Printf("Logger: Sistema de logging inicializado - archivo: %s\n", logPath)
	return nil
}

// WriteLog guarda una entrada de log en formato JSON por línea
func WriteLog(entry LogEntry) error {
	logMutex.Lock()
	defer logMutex.Unlock()

	// Verificar que el logger esté inicializado
	if logFile == nil {
		return fmt.Errorf("logger no inicializado - llama InitLogger() primero")
	}

	// Añadir timestamp de creación
	entry.CreatedAt = time.Now().Format("2006-01-02 15:04:05.000")

	// Convertir a JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("error serializando log entry: %v", err)
	}

	// Escribir al archivo (una línea por entrada)
	_, err = logFile.Write(jsonData)
	if err != nil {
		return fmt.Errorf("error escribiendo al archivo de log: %v", err)
	}

	// Añadir nueva línea
	_, err = logFile.Write([]byte("\n"))
	if err != nil {
		return fmt.Errorf("error escribiendo nueva línea: %v", err)
	}

	// Forzar escritura al disco
	logFile.Sync()

	return nil
}

// LogSchedulerEvent registra un evento del scheduler
func LogSchedulerEvent(eventType string, timestamp int, action string, lvt int, message string) {
	entry := LogEntry{
		ThreadType: "Scheduler",
		ThreadID:   0,
		EventType:  eventType,
		Timestamp:  timestamp,
		Action:     action,
		LVT:        lvt,
		Message:    message,
	}

	if err := WriteLog(entry); err != nil {
		fmt.Printf("Error escribiendo log del scheduler: %v\n", err)
	}
}

// LogWorkerEvent registra un evento de un worker
func LogWorkerEvent(workerID int, eventType string, timestamp int, action string, lvt int, message string) {
	entry := LogEntry{
		ThreadType: "Worker",
		ThreadID:   workerID,
		EventType:  eventType,
		Timestamp:  timestamp,
		Action:     action,
		LVT:        lvt,
		Message:    message,
	}

	if err := WriteLog(entry); err != nil {
		fmt.Printf("Error escribiendo log del worker %d: %v\n", workerID, err)
	}
}

// LogSend registra el envío de un evento
func LogSend(senderType string, senderID int, eventType string, timestamp int, lvt int, targetID int) {
	message := fmt.Sprintf("Enviando evento %s a %s %d", eventType,
		map[bool]string{true: "Worker", false: "Scheduler"}[senderType == "Scheduler"], targetID)

	entry := LogEntry{
		ThreadType: senderType,
		ThreadID:   senderID,
		EventType:  eventType,
		Timestamp:  timestamp,
		Action:     "Send",
		LVT:        lvt,
		Message:    message,
	}

	if err := WriteLog(entry); err != nil {
		fmt.Printf("Error escribiendo log de envío: %v\n", err)
	}
}

// LogReceive registra la recepción de un evento
func LogReceive(receiverType string, receiverID int, eventType string, timestamp int, lvt int, senderID int) {
	message := fmt.Sprintf("Recibiendo evento %s de %s %d", eventType,
		map[bool]string{true: "Scheduler", false: "Worker"}[receiverType == "Worker"], senderID)

	entry := LogEntry{
		ThreadType: receiverType,
		ThreadID:   receiverID,
		EventType:  eventType,
		Timestamp:  timestamp,
		Action:     "Receive",
		LVT:        lvt,
		Message:    message,
	}

	if err := WriteLog(entry); err != nil {
		fmt.Printf("Error escribiendo log de recepción: %v\n", err)
	}
}

// LogCheckpoint registra la creación de un checkpoint
func LogCheckpoint(workerID int, timestamp int, lvt int, eventCount int) {
	message := fmt.Sprintf("Checkpoint creado - EventCount: %d", eventCount)

	entry := LogEntry{
		ThreadType: "Worker",
		ThreadID:   workerID,
		EventType:  "CHECKPOINT",
		Timestamp:  timestamp,
		Action:     "Checkpoint",
		LVT:        lvt,
		Message:    message,
	}

	if err := WriteLog(entry); err != nil {
		fmt.Printf("Error escribiendo log de checkpoint: %v\n", err)
	}
}

// LogRollback registra un rollback
func LogRollback(workerID int, targetTimestamp int, lvt int, stragglerTimestamp int) {
	message := fmt.Sprintf("Rollback a timestamp %d debido a straggler en %d",
		targetTimestamp, stragglerTimestamp)

	entry := LogEntry{
		ThreadType: "Worker",
		ThreadID:   workerID,
		EventType:  "ROLLBACK",
		Timestamp:  targetTimestamp,
		Action:     "Rollback",
		LVT:        lvt,
		Message:    message,
	}

	if err := WriteLog(entry); err != nil {
		fmt.Printf("Error escribiendo log de rollback: %v\n", err)
	}
}

// LogStraggler registra la detección de un straggler
func LogStraggler(workerID int, stragglerTimestamp int, currentLVT int, eventID int) {
	message := fmt.Sprintf("STRAGGLER DETECTADO - Evento ID:%d timestamp:%d < LVT:%d",
		eventID, stragglerTimestamp, currentLVT)

	entry := LogEntry{
		ThreadType: "Worker",
		ThreadID:   workerID,
		EventType:  "STRAGGLER",
		Timestamp:  stragglerTimestamp,
		Action:     "Detect",
		LVT:        currentLVT,
		Message:    message,
	}

	if err := WriteLog(entry); err != nil {
		fmt.Printf("Error escribiendo log de straggler: %v\n", err)
	}
}

// CloseLogger cierra el archivo de log de manera segura
func CloseLogger() error {
	logMutex.Lock()
	defer logMutex.Unlock()

	if logFile != nil {
		// Escribir entrada de cierre
		closeEntry := LogEntry{
			ThreadType: "System",
			ThreadID:   0,
			EventType:  "SHUTDOWN",
			Timestamp:  0,
			Action:     "Stop",
			LVT:        0,
			Message:    "Simulador terminado - Cerrando logs",
			CreatedAt:  time.Now().Format("2006-01-02 15:04:05.000"),
		}

		closeJSON, _ := json.Marshal(closeEntry)
		logFile.Write(closeJSON)
		logFile.Write([]byte("\n"))
		logFile.Sync()

		err := logFile.Close()
		logFile = nil

		fmt.Printf("Logger: Sistema de logging cerrado\n")
		return err
	}

	return nil
}

// ReadLogs lee y retorna todas las entradas del log
func ReadLogs() ([]LogEntry, error) {
	file, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("error abriendo archivo de log %s: %v", logPath, err)
	}
	defer file.Close()

	var entries []LogEntry
	decoder := json.NewDecoder(file)

	for decoder.More() {
		var entry LogEntry
		if err := decoder.Decode(&entry); err != nil {
			// Ignorar líneas malformadas y continuar
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// SetLogPath permite cambiar la ruta del archivo de log
func SetLogPath(path string) {
	logMutex.Lock()
	defer logMutex.Unlock()

	logPath = path
}

// GetLogPath retorna la ruta actual del archivo de log
func GetLogPath() string {
	logMutex.Lock()
	defer logMutex.Unlock()

	return logPath
}
