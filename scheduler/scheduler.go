package scheduler

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"certamen2/event"
	"certamen2/worker"
)

// Scheduler representa el hilo principal que coordina la simulación
type Scheduler struct {
	LVT            int              // Local Virtual Time del scheduler
	Workers        []*worker.Worker // Lista de workers disponibles
	EventIDCounter int              // Contador para generar IDs únicos de eventos
}

// NewScheduler crea un nuevo scheduler con los workers especificados
func NewScheduler(workers []*worker.Worker) *Scheduler {
	return &Scheduler{
		LVT:            0,
		Workers:        workers,
		EventIDCounter: 0,
	}
}

// generateExternalEvent genera un evento externo con timestamp creciente
func (s *Scheduler) generateExternalEvent(targetWorkerID int) event.Event {
	// Generar step aleatorio entre 1 y 5 para mantener causalidad
	randomStep := rand.Intn(5) + 1
	s.LVT += randomStep

	// Incrementar contador de ID
	s.EventIDCounter++

	// Crear evento externo
	e := event.NewEvent(s.EventIDCounter, event.EXTERNAL, s.LVT, targetWorkerID)

	log.Printf("Scheduler: Generado evento externo %s para Worker %d", e.String(), targetWorkerID)
	return e
}

// dispatchEvents envía eventos externos a los Workers por canal
func (s *Scheduler) dispatchEvents(events []event.Event) {
	log.Printf("Scheduler: Despachando %d eventos", len(events))

	for _, e := range events {
		// Buscar el worker de destino
		var targetWorker *worker.Worker
		for _, w := range s.Workers {
			if w.ID == e.WorkerID {
				targetWorker = w
				break
			}
		}

		if targetWorker != nil {
			// Enviar evento al worker correspondiente
			select {
			case targetWorker.InChannel <- e:
				log.Printf("Scheduler: Evento %s enviado exitosamente a Worker %d",
					e.String(), e.WorkerID)
			default:
				log.Printf("Scheduler: ADVERTENCIA - Canal del Worker %d está lleno, evento perdido: %s",
					e.WorkerID, e.String())
			}
		} else {
			log.Printf("Scheduler: ERROR - Worker %d no encontrado para evento %s",
				e.WorkerID, e.String())
		}
	}
}

// generateRandomEvents genera una lista de eventos aleatorios
func (s *Scheduler) generateRandomEvents(numEvents int) []event.Event {
	events := make([]event.Event, 0, numEvents)

	for i := 0; i < numEvents; i++ {
		// Seleccionar worker de destino aleatoriamente
		targetWorkerID := rand.Intn(len(s.Workers))
		if len(s.Workers) > 0 {
			targetWorkerID = s.Workers[rand.Intn(len(s.Workers))].ID
		}

		// Generar evento externo
		e := s.generateExternalEvent(targetWorkerID)
		events = append(events, e)
	}

	return events
}

// Run ejecuta el loop principal del scheduler
func (s *Scheduler) Run(iterations int, eventsPerIteration int, sleepDuration time.Duration) {
	log.Printf("Scheduler: Iniciando simulación - %d iteraciones, %d eventos por iteración",
		iterations, eventsPerIteration)

	// Inicializar generador de números aleatorios
	rand.Seed(time.Now().UnixNano())

	// Iniciar todos los workers en goroutines
	for _, w := range s.Workers {
		go w.Run()
		log.Printf("Scheduler: Worker %d iniciado", w.ID)
	}

	// Loop principal de la simulación
	for i := 0; i < iterations; i++ {
		log.Printf("Scheduler: === ITERACIÓN %d ===", i+1)

		// Generar eventos para esta iteración
		events := s.generateRandomEvents(eventsPerIteration)

		// Despachar eventos a los workers
		s.dispatchEvents(events)

		// Mostrar estado actual del scheduler
		log.Printf("Scheduler: LVT actual: %d, Eventos generados: %d",
			s.LVT, s.EventIDCounter)

		// Mostrar estado de todos los workers
		for _, w := range s.Workers {
			log.Printf("Scheduler: %s", w.GetStatus())
		}

		// Esperar antes de la siguiente iteración
		if i < iterations-1 { // No esperar después de la última iteración
			log.Printf("Scheduler: Esperando %v antes de la siguiente iteración", sleepDuration)
			time.Sleep(sleepDuration)
		}
	}

	// Esperar un poco más para que los workers procesen todos los eventos
	log.Printf("Scheduler: Esperando que los workers procesen los eventos restantes...")
	time.Sleep(sleepDuration * 2)

	// Cerrar todos los workers
	s.stopAllWorkers()

	log.Printf("Scheduler: Simulación completada - Total eventos generados: %d, LVT final: %d",
		s.EventIDCounter, s.LVT)
}

// stopAllWorkers detiene todos los workers de manera ordenada
func (s *Scheduler) stopAllWorkers() {
	log.Printf("Scheduler: Deteniendo todos los workers...")

	for _, w := range s.Workers {
		w.Stop()
		log.Printf("Scheduler: Worker %d detenido", w.ID)
	}
}

// GetStatus devuelve información sobre el estado actual del scheduler
func (s *Scheduler) GetStatus() string {
	return fmt.Sprintf("Scheduler: LVT=%d, EventsGenerated=%d, Workers=%d",
		s.LVT, s.EventIDCounter, len(s.Workers))
}

// AddWorker añade un nuevo worker al scheduler
func (s *Scheduler) AddWorker(w *worker.Worker) {
	s.Workers = append(s.Workers, w)
	log.Printf("Scheduler: Worker %d añadido al scheduler", w.ID)
}

// RemoveWorker elimina un worker del scheduler
func (s *Scheduler) RemoveWorker(workerID int) bool {
	for i, w := range s.Workers {
		if w.ID == workerID {
			// Detener el worker antes de eliminarlo
			w.Stop()

			// Eliminar del slice
			s.Workers = append(s.Workers[:i], s.Workers[i+1:]...)
			log.Printf("Scheduler: Worker %d eliminado del scheduler", workerID)
			return true
		}
	}

	log.Printf("Scheduler: Worker %d no encontrado para eliminar", workerID)
	return false
}
