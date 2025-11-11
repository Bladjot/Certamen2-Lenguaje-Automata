package event

import "fmt"

// Constantes para tipos de evento
const (
	EXTERNAL = "EXTERNAL"
	INTERNAL = "INTERNAL"
)

// Event representa un evento en el simulador distribuido
type Event struct {
	ID        int    // Identificador único del evento
	Type      string // Tipo de evento (EXTERNAL o INTERNAL)
	Timestamp int    // Marca temporal del evento
	WorkerID  int    // ID del worker que procesa el evento
}

// NewEvent crea un nuevo evento con los parámetros especificados
func NewEvent(id int, eventType string, timestamp int, workerID int) Event {
	return Event{
		ID:        id,
		Type:      eventType,
		Timestamp: timestamp,
		WorkerID:  workerID,
	}
}

// String devuelve una representación legible del evento
func (e Event) String() string {
	return fmt.Sprintf("Event{ID: %d, Type: %s, Timestamp: %d, WorkerID: %d}",
		e.ID, e.Type, e.Timestamp, e.WorkerID)
}

// Compare compara dos eventos según su orden temporal
// Devuelve -1 si e1 < e2, 0 si e1 == e2, 1 si e1 > e2
func Compare(e1, e2 Event) int {
	if e1.Timestamp < e2.Timestamp {
		return -1
	}
	if e1.Timestamp > e2.Timestamp {
		return 1
	}
	// Si los timestamps son iguales, comparar por ID para desempatar
	if e1.ID < e2.ID {
		return -1
	}
	if e1.ID > e2.ID {
		return 1
	}
	return 0
}
