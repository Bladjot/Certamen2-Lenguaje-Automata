package worker

import (
	"fmt"
	"log"
	"sync"

	"certamen2/event"
)

// StateSnapshot representa un snapshot del estado del worker
type StateSnapshot struct {
	LVT       int           // Local Virtual Time en el momento del checkpoint
	EventsLog []event.Event // Copia de los eventos procesados hasta el checkpoint
	Timestamp int           // Timestamp del checkpoint
}

// Worker representa un worker en el simulador distribuido
type Worker struct {
	ID          int                   // Identificador único del worker
	LVT         int                   // Local Virtual Time
	InChannel   chan event.Event      // Canal para recibir eventos
	History     []event.Event         // Historial de eventos procesados
	Checkpoints map[int]StateSnapshot // Checkpoints del estado
	mu          sync.Mutex            // Mutex para proteger secciones críticas
}

// NewWorker crea un nuevo worker con los parámetros especificados
func NewWorker(id int) *Worker {
	return &Worker{
		ID:          id,
		LVT:         0,
		InChannel:   make(chan event.Event, 100), // Buffer para eventos
		History:     make([]event.Event, 0),
		Checkpoints: make(map[int]StateSnapshot),
		mu:          sync.Mutex{},
	}
}

// Run es la goroutine principal que espera eventos desde InChannel y los procesa
func (w *Worker) Run() {
	log.Printf("Worker %d: Iniciando goroutine principal", w.ID)

	for e := range w.InChannel {
		log.Printf("Worker %d: Recibido evento %s", w.ID, e.String())
		w.processEvent(e)
	}

	log.Printf("Worker %d: Terminando goroutine principal", w.ID)
}

// processEvent procesa un evento según las reglas de Time Warp
func (w *Worker) processEvent(e event.Event) {
	w.mu.Lock()
	defer w.mu.Unlock()

	log.Printf("Worker %d: Procesando evento %s (LVT actual: %d)", w.ID, e.String(), w.LVT)

	if e.Timestamp >= w.LVT {
		// Evento en orden correcto - procesar normalmente
		log.Printf("Worker %d: Evento en orden correcto, procesando", w.ID)
		w.executeEvent(e)
		w.LVT = e.Timestamp + 1 // Avanzar LVT
		w.History = append(w.History, e)

		// Crear checkpoint periódicamente
		if len(w.History)%5 == 0 {
			w.createCheckpoint(e)
		}

		log.Printf("Worker %d: Evento procesado, nuevo LVT: %d", w.ID, w.LVT)
	} else {
		// Straggler detectado - necesita rollback
		log.Printf("Worker %d: STRAGGLER DETECTADO! Evento timestamp: %d < LVT: %d",
			w.ID, e.Timestamp, w.LVT)
		w.rollback(e.Timestamp)

		// Procesar el evento straggler
		w.executeEvent(e)

		// Insertar en el lugar correcto del historial
		w.insertEventInHistory(e)

		// Regenerar estado desde el rollback
		w.regenerateState()
	}
}

// executeEvent simula la ejecución de un evento
func (w *Worker) executeEvent(e event.Event) {
	// Aquí iría la lógica específica de procesamiento del evento
	// Por ahora, solo simulamos con un log
	log.Printf("Worker %d: Ejecutando evento ID:%d Type:%s", w.ID, e.ID, e.Type)
}

// createCheckpoint guarda un snapshot del estado actual
func (w *Worker) createCheckpoint(e event.Event) {
	log.Printf("Worker %d: Creando checkpoint en timestamp %d", w.ID, e.Timestamp)

	// Crear copia del historial actual
	historyCopy := make([]event.Event, len(w.History))
	copy(historyCopy, w.History)

	snapshot := StateSnapshot{
		LVT:       w.LVT,
		EventsLog: historyCopy,
		Timestamp: e.Timestamp,
	}

	w.Checkpoints[e.Timestamp] = snapshot
	log.Printf("Worker %d: Checkpoint creado exitosamente para timestamp %d", w.ID, e.Timestamp)
}

// rollback restaura el estado a un timestamp anterior
func (w *Worker) rollback(targetTimestamp int) {
	log.Printf("Worker %d: Iniciando rollback a timestamp %d", w.ID, targetTimestamp)

	// Encontrar el checkpoint más reciente anterior al targetTimestamp
	var bestCheckpoint StateSnapshot
	var bestTimestamp int = -1

	for timestamp, checkpoint := range w.Checkpoints {
		if timestamp <= targetTimestamp && timestamp > bestTimestamp {
			bestCheckpoint = checkpoint
			bestTimestamp = timestamp
		}
	}

	if bestTimestamp != -1 {
		// Restaurar desde checkpoint
		log.Printf("Worker %d: Restaurando desde checkpoint timestamp %d", w.ID, bestTimestamp)
		w.LVT = bestCheckpoint.LVT
		w.History = make([]event.Event, len(bestCheckpoint.EventsLog))
		copy(w.History, bestCheckpoint.EventsLog)
	} else {
		// No hay checkpoint disponible, reiniciar desde el principio
		log.Printf("Worker %d: No hay checkpoint disponible, reiniciando desde el principio", w.ID)
		w.LVT = 0
		w.History = make([]event.Event, 0)
	}

	// Eliminar checkpoints posteriores al targetTimestamp
	for timestamp := range w.Checkpoints {
		if timestamp > targetTimestamp {
			delete(w.Checkpoints, timestamp)
			log.Printf("Worker %d: Eliminado checkpoint timestamp %d", w.ID, timestamp)
		}
	}

	log.Printf("Worker %d: Rollback completado, LVT restaurado a %d", w.ID, w.LVT)
}

// insertEventInHistory inserta un evento en el lugar correcto del historial
func (w *Worker) insertEventInHistory(e event.Event) {
	// Encontrar la posición correcta para insertar
	insertPos := 0
	for i, histEvent := range w.History {
		if event.Compare(e, histEvent) <= 0 {
			insertPos = i
			break
		}
		insertPos = i + 1
	}

	// Insertar en la posición correcta
	w.History = append(w.History, event.Event{})
	copy(w.History[insertPos+1:], w.History[insertPos:])
	w.History[insertPos] = e

	log.Printf("Worker %d: Evento insertado en posición %d del historial", w.ID, insertPos)
}

// regenerateState reconstruye el estado reprocesando eventos desde el rollback
func (w *Worker) regenerateState() {
	log.Printf("Worker %d: Regenerando estado desde historial", w.ID)

	// Reprocesar todos los eventos en el historial
	for i, e := range w.History {
		if e.Timestamp >= w.LVT {
			log.Printf("Worker %d: Reprocesando evento %d: %s", w.ID, i, e.String())
			w.executeEvent(e)
			w.LVT = e.Timestamp + 1

			// Crear checkpoint periódicamente durante la regeneración
			if (i+1)%5 == 0 {
				w.createCheckpoint(e)
			}
		}
	}

	log.Printf("Worker %d: Estado regenerado completamente, LVT final: %d", w.ID, w.LVT)
}

// Stop cierra el canal de entrada del worker
func (w *Worker) Stop() {
	log.Printf("Worker %d: Cerrando canal de entrada", w.ID)
	close(w.InChannel)
}

// GetStatus devuelve información sobre el estado actual del worker
func (w *Worker) GetStatus() string {
	w.mu.Lock()
	defer w.mu.Unlock()

	return fmt.Sprintf("Worker %d: LVT=%d, Events=%d, Checkpoints=%d",
		w.ID, w.LVT, len(w.History), len(w.Checkpoints))
}
