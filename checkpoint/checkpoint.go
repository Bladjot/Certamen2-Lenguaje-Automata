package checkpoint

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// StateSnapshot representa un snapshot del estado de un worker
type StateSnapshot struct {
	WorkerID   int            // ID del worker
	Timestamp  int            // Timestamp del checkpoint
	Variables  map[string]int // Variables de estado del worker
	LVT        int            // Local Virtual Time en el momento del checkpoint
	EventCount int            // Número de eventos procesados hasta el checkpoint
}

// NewStateSnapshot crea un nuevo snapshot con los parámetros especificados
func NewStateSnapshot(workerID int, timestamp int, variables map[string]int, lvt int, eventCount int) StateSnapshot {
	// Crear copia del mapa de variables para evitar referencias compartidas
	varCopy := make(map[string]int)
	for k, v := range variables {
		varCopy[k] = v
	}

	return StateSnapshot{
		WorkerID:   workerID,
		Timestamp:  timestamp,
		Variables:  varCopy,
		LVT:        lvt,
		EventCount: eventCount,
	}
}

// SaveCheckpoint guarda un snapshot en un archivo usando encoding/gob
func SaveCheckpoint(workerID int, snapshot StateSnapshot, path string) error {
	// Crear el directorio si no existe
	err := os.MkdirAll(path, 0755)
	if err != nil {
		log.Printf("Error creando directorio %s: %v", path, err)
		return fmt.Errorf("error creando directorio %s: %v", path, err)
	}

	// Generar nombre del archivo: worker_<id>_t<timestamp>.chk
	filename := fmt.Sprintf("worker_%d_t%d.chk", workerID, snapshot.Timestamp)
	fullPath := filepath.Join(path, filename)

	log.Printf("Checkpoint: Guardando snapshot de Worker %d en %s", workerID, fullPath)

	// Crear el archivo
	file, err := os.Create(fullPath)
	if err != nil {
		log.Printf("Error creando archivo %s: %v", fullPath, err)
		return fmt.Errorf("error creando archivo %s: %v", fullPath, err)
	}
	defer file.Close()

	// Crear encoder gob y guardar el snapshot
	encoder := gob.NewEncoder(file)
	err = encoder.Encode(snapshot)
	if err != nil {
		log.Printf("Error codificando snapshot: %v", err)
		return fmt.Errorf("error codificando snapshot: %v", err)
	}

	log.Printf("Checkpoint: Snapshot guardado exitosamente - Worker %d, Timestamp %d",
		workerID, snapshot.Timestamp)
	return nil
}

// LoadCheckpoint carga un snapshot desde un archivo usando encoding/gob
func LoadCheckpoint(workerID int, timestamp int, path string) (StateSnapshot, error) {
	// Generar nombre del archivo
	filename := fmt.Sprintf("worker_%d_t%d.chk", workerID, timestamp)
	fullPath := filepath.Join(path, filename)

	log.Printf("Checkpoint: Cargando snapshot de Worker %d desde %s", workerID, fullPath)

	// Verificar que el archivo existe
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		log.Printf("Checkpoint: Archivo no encontrado %s", fullPath)
		return StateSnapshot{}, fmt.Errorf("checkpoint no encontrado: %s", fullPath)
	}

	// Abrir el archivo
	file, err := os.Open(fullPath)
	if err != nil {
		log.Printf("Error abriendo archivo %s: %v", fullPath, err)
		return StateSnapshot{}, fmt.Errorf("error abriendo archivo %s: %v", fullPath, err)
	}
	defer file.Close()

	// Crear decoder gob y cargar el snapshot
	var snapshot StateSnapshot
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&snapshot)
	if err != nil {
		log.Printf("Error decodificando snapshot: %v", err)
		return StateSnapshot{}, fmt.Errorf("error decodificando snapshot: %v", err)
	}

	log.Printf("Checkpoint: Snapshot cargado exitosamente - Worker %d, Timestamp %d",
		workerID, snapshot.Timestamp)
	return snapshot, nil
}

// LoadLatestCheckpoint carga el checkpoint más reciente para un worker
func LoadLatestCheckpoint(workerID int, path string) (StateSnapshot, error) {
	log.Printf("Checkpoint: Buscando último checkpoint para Worker %d en %s", workerID, path)

	// Leer el directorio
	files, err := os.ReadDir(path)
	if err != nil {
		log.Printf("Error leyendo directorio %s: %v", path, err)
		return StateSnapshot{}, fmt.Errorf("error leyendo directorio %s: %v", path, err)
	}

	// Buscar archivos de checkpoint para este worker
	var latestTimestamp int = -1
	var latestSnapshot StateSnapshot

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Verificar si es un archivo de checkpoint para este worker
		if strings.HasPrefix(file.Name(), fmt.Sprintf("worker_%d_t", workerID)) &&
			strings.HasSuffix(file.Name(), ".chk") {

			// Extraer timestamp del nombre del archivo
			parts := strings.Split(file.Name(), "_")
			if len(parts) >= 3 {
				timestampPart := strings.TrimSuffix(parts[2], ".chk")
				timestampPart = strings.TrimPrefix(timestampPart, "t")

				if timestamp, err := strconv.Atoi(timestampPart); err == nil {
					if timestamp > latestTimestamp {
						// Cargar este checkpoint
						if snapshot, err := LoadCheckpoint(workerID, timestamp, path); err == nil {
							latestTimestamp = timestamp
							latestSnapshot = snapshot
						}
					}
				}
			}
		}
	}

	if latestTimestamp == -1 {
		log.Printf("Checkpoint: No se encontraron checkpoints para Worker %d", workerID)
		return StateSnapshot{}, fmt.Errorf("no se encontraron checkpoints para Worker %d", workerID)
	}

	log.Printf("Checkpoint: Checkpoint más reciente encontrado - Worker %d, Timestamp %d",
		workerID, latestTimestamp)
	return latestSnapshot, nil
}

// ListCheckpoints lista todos los checkpoints disponibles para un worker
func ListCheckpoints(workerID int, path string) ([]int, error) {
	log.Printf("Checkpoint: Listando checkpoints para Worker %d en %s", workerID, path)

	// Leer el directorio
	files, err := os.ReadDir(path)
	if err != nil {
		log.Printf("Error leyendo directorio %s: %v", path, err)
		return nil, fmt.Errorf("error leyendo directorio %s: %v", path, err)
	}

	var timestamps []int

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Verificar si es un archivo de checkpoint para este worker
		if strings.HasPrefix(file.Name(), fmt.Sprintf("worker_%d_t", workerID)) &&
			strings.HasSuffix(file.Name(), ".chk") {

			// Extraer timestamp del nombre del archivo
			parts := strings.Split(file.Name(), "_")
			if len(parts) >= 3 {
				timestampPart := strings.TrimSuffix(parts[2], ".chk")
				timestampPart = strings.TrimPrefix(timestampPart, "t")

				if timestamp, err := strconv.Atoi(timestampPart); err == nil {
					timestamps = append(timestamps, timestamp)
				}
			}
		}
	}

	log.Printf("Checkpoint: Encontrados %d checkpoints para Worker %d", len(timestamps), workerID)
	return timestamps, nil
}

// DeleteCheckpoint elimina un checkpoint específico
func DeleteCheckpoint(workerID int, timestamp int, path string) error {
	filename := fmt.Sprintf("worker_%d_t%d.chk", workerID, timestamp)
	fullPath := filepath.Join(path, filename)

	log.Printf("Checkpoint: Eliminando checkpoint %s", fullPath)

	err := os.Remove(fullPath)
	if err != nil {
		log.Printf("Error eliminando checkpoint %s: %v", fullPath, err)
		return fmt.Errorf("error eliminando checkpoint %s: %v", fullPath, err)
	}

	log.Printf("Checkpoint: Checkpoint eliminado exitosamente")
	return nil
}

// DeleteAllCheckpoints elimina todos los checkpoints de un worker
func DeleteAllCheckpoints(workerID int, path string) error {
	log.Printf("Checkpoint: Eliminando todos los checkpoints para Worker %d", workerID)

	timestamps, err := ListCheckpoints(workerID, path)
	if err != nil {
		return err
	}

	for _, timestamp := range timestamps {
		if err := DeleteCheckpoint(workerID, timestamp, path); err != nil {
			log.Printf("Error eliminando checkpoint timestamp %d: %v", timestamp, err)
			// Continuar eliminando los demás checkpoints
		}
	}

	log.Printf("Checkpoint: Eliminados todos los checkpoints para Worker %d", workerID)
	return nil
}

// String devuelve una representación legible del snapshot
func (s StateSnapshot) String() string {
	return fmt.Sprintf("StateSnapshot{WorkerID: %d, Timestamp: %d, LVT: %d, EventCount: %d, Variables: %v}",
		s.WorkerID, s.Timestamp, s.LVT, s.EventCount, s.Variables)
}
