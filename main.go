package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"sync"
	"time"
)

// Tipo de evento
const (
	TipoExterno = iota
	TipoInterno
)

// Evento representa un evento externo generado por el scheduler.
type Evento struct {
	ID              int `json:"event_id"`
	Tipo            int `json:"tipo"`
	Tiempo          int `json:"timestamp"`
	WorkerDestinoID int `json:"worker_id"`
}

// WorkerState modela las variables que deben sobrevivir a un rollback.
type WorkerState struct {
	LVT int `json:"lvt"`
}

// Checkpoint almacena el estado y el número de eventos procesados.
type Checkpoint struct {
	Estado     WorkerState
	HistoryLen int
}

// SimulationConfig parametriza la ejecución.
type SimulationConfig struct {
	NumWorkers          int
	TotalExternalEvents int
	InternalMinEvents   int
	InternalMaxEvents   int
	InternalMinJump     int
	InternalMaxJump     int
	ChannelBuffer       int
	LogPath             string
	Seed                int64
}

// SimulationResult resume los datos claves de una corrida.
type SimulationResult struct {
	Duration         time.Duration
	EventsDispatched int
	WorkerStats      []WorkerStats
}

// WorkerStats expone métricas por worker.
type WorkerStats struct {
	ID               int
	ExternalEvents   int
	InternalEvents   int
	Rollbacks        int
	LastVirtualTime  int
	CheckpointsBuilt int
}

// Scheduler genera eventos externos en orden creciente de tiempo.
type Scheduler struct {
	cfg         SimulationConfig
	workerChans []chan Evento
	logger      *Logger
	rand        *rand.Rand
	clock       int
	nextEventID int
}

func NewScheduler(cfg SimulationConfig, workerChans []chan Evento, logger *Logger) *Scheduler {
	return &Scheduler{
		cfg:         cfg,
		workerChans: workerChans,
		logger:      logger,
		rand:        rand.New(rand.NewSource(cfg.Seed + 42)),
	}
}

func (s *Scheduler) Run() {
	if s.cfg.TotalExternalEvents == 0 {
		return
	}
	for i := 0; i < s.cfg.TotalExternalEvents; i++ {
		s.clock += s.rand.Intn(4) + 1
		target := s.rand.Intn(s.cfg.NumWorkers)
		event := Evento{
			ID:              s.nextEventID,
			Tipo:            TipoExterno,
			Tiempo:          s.clock,
			WorkerDestinoID: target,
		}
		s.nextEventID++
		s.logger.Log(LogEntry{
			WallTime: time.Now(),
			Entity:   "scheduler",
			Event:    "external_dispatched",
			SimTime:  s.clock,
			EventID:  event.ID,
			Target:   target,
		})
		s.workerChans[target] <- event
	}
}

// Worker procesa eventos y maneja checkpoints/rollbacks.
type Worker struct {
	id          int
	cfg         SimulationConfig
	input       chan Evento
	logger      *Logger
	rand        *rand.Rand
	state       WorkerState
	history     []Evento
	checkpoints []Checkpoint
	stats       WorkerStats
}

func NewWorker(id int, cfg SimulationConfig, input chan Evento, logger *Logger) *Worker {
	w := &Worker{
		id:     id,
		cfg:    cfg,
		input:  input,
		logger: logger,
		rand:   rand.New(rand.NewSource(cfg.Seed + int64(id)*17 + 99)),
		state:  WorkerState{LVT: 0},
	}
	w.stats.ID = id
	w.checkpoints = append(w.checkpoints, Checkpoint{Estado: w.state, HistoryLen: 0})
	return w
}

func (w *Worker) Run(wg *sync.WaitGroup) {
	defer wg.Done()
	for event := range w.input {
		w.handleExternal(event)
	}
	w.logger.Log(LogEntry{
		WallTime: time.Now(),
		Entity:   w.entityName(),
		Event:    "worker_stopped",
		WorkerID: w.id,
		SimTime:  w.state.LVT,
	})
}

func (w *Worker) entityName() string {
	return fmt.Sprintf("worker-%d", w.id)
}

func (w *Worker) handleExternal(event Evento) {
	w.logger.Log(LogEntry{
		WallTime: time.Now(),
		Entity:   w.entityName(),
		Event:    "external_received",
		WorkerID: w.id,
		EventID:  event.ID,
		SimTime:  w.state.LVT,
		Details: map[string]any{
			"event_timestamp": event.Tiempo,
		},
	})
	w.createCheckpoint(len(w.history), "live")
	if event.Tiempo < w.state.LVT {
		w.logger.Log(LogEntry{
			WallTime: time.Now(),
			Entity:   w.entityName(),
			Event:    "straggler_detected",
			WorkerID: w.id,
			EventID:  event.ID,
			SimTime:  w.state.LVT,
			Details: map[string]any{
				"event_timestamp": event.Tiempo,
			},
		})
		w.performRollback(event)
		return
	}
	w.appendEvent(event)
	w.processEvent(event, false)
}

func (w *Worker) appendEvent(event Evento) {
	idx := sort.Search(len(w.history), func(i int) bool {
		if w.history[i].Tiempo == event.Tiempo {
			return w.history[i].ID >= event.ID
		}
		return w.history[i].Tiempo > event.Tiempo
	})
	w.history = append(w.history, Evento{})
	copy(w.history[idx+1:], w.history[idx:])
	w.history[idx] = event
}

func (w *Worker) processEvent(event Evento, fromReplay bool) {
	prev := w.state.LVT
	w.state.LVT = event.Tiempo
	w.stats.ExternalEvents++
	w.logger.Log(LogEntry{
		WallTime: time.Now(),
		Entity:   w.entityName(),
		Event:    "external_processed",
		WorkerID: w.id,
		EventID:  event.ID,
		SimTime:  w.state.LVT,
		Details: map[string]any{
			"from_replay":  fromReplay,
			"previous_lvt": prev,
		},
	})
	w.generateInternalEvents()
	w.stats.LastVirtualTime = w.state.LVT
}

func (w *Worker) generateInternalEvents() {
	count := w.rand.Intn(w.cfg.InternalMaxEvents-w.cfg.InternalMinEvents+1) + w.cfg.InternalMinEvents
	for i := 0; i < count; i++ {
		jump := w.rand.Intn(w.cfg.InternalMaxJump-w.cfg.InternalMinJump+1) + w.cfg.InternalMinJump
		prev := w.state.LVT
		w.state.LVT += jump
		w.stats.InternalEvents++
		w.logger.Log(LogEntry{
			WallTime: time.Now(),
			Entity:   w.entityName(),
			Event:    "internal_processed",
			WorkerID: w.id,
			SimTime:  w.state.LVT,
			Details: map[string]any{
				"previous_lvt": prev,
				"jump":         jump,
			},
		})
	}
}

func (w *Worker) performRollback(straggler Evento) {
	w.stats.Rollbacks++
	w.appendEvent(straggler)
	cpIdx := w.findCheckpointFor(straggler.Tiempo)
	if cpIdx < 0 {
		cpIdx = 0
	}
	rollbackFrom := w.state.LVT
	cp := w.checkpoints[cpIdx]
	w.logger.Log(LogEntry{
		WallTime:     time.Now(),
		Entity:       w.entityName(),
		Event:        "rollback_start",
		WorkerID:     w.id,
		SimTime:      cp.Estado.LVT,
		RollbackFrom: rollbackFrom,
		RollbackTo:   straggler.Tiempo,
	})
	w.state = cp.Estado
	w.stats.LastVirtualTime = w.state.LVT
	w.checkpoints = w.checkpoints[:cpIdx+1]
	for idx := cp.HistoryLen; idx < len(w.history); idx++ {
		e := w.history[idx]
		w.createCheckpoint(idx, "replay")
		w.processEvent(e, true)
	}
	w.logger.Log(LogEntry{
		WallTime:     time.Now(),
		Entity:       w.entityName(),
		Event:        "rollback_end",
		WorkerID:     w.id,
		SimTime:      w.state.LVT,
		RollbackFrom: rollbackFrom,
		RollbackTo:   straggler.Tiempo,
	})
}

func (w *Worker) findCheckpointFor(ts int) int {
	for i := len(w.checkpoints) - 1; i >= 0; i-- {
		if w.checkpoints[i].Estado.LVT <= ts {
			return i
		}
	}
	return -1
}

func (w *Worker) createCheckpoint(historyLen int, mode string) {
	cp := Checkpoint{Estado: w.state, HistoryLen: historyLen}
	w.checkpoints = append(w.checkpoints, cp)
	w.stats.CheckpointsBuilt++
	w.logger.Log(LogEntry{
		WallTime: time.Now(),
		Entity:   w.entityName(),
		Event:    "checkpoint_created",
		WorkerID: w.id,
		SimTime:  w.state.LVT,
		Details: map[string]any{
			"history_len": historyLen,
			"mode":        mode,
		},
	})
}

// Logger maneja la escritura concurrente de entradas JSON line-by-line.
type Logger struct {
	mu   sync.Mutex
	enc  *json.Encoder
	file *os.File
}

func NewLogger(path string) (*Logger, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	enc := json.NewEncoder(file)
	return &Logger{enc: enc, file: file}, nil
}

func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

func (l *Logger) Log(entry LogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if entry.WallTime.IsZero() {
		entry.WallTime = time.Now()
	}
	if err := l.enc.Encode(entry); err != nil {
		fmt.Fprintf(os.Stderr, "logger error: %v\n", err)
	}
}

// LogEntry describe los eventos significativos a registrar.
type LogEntry struct {
	WallTime     time.Time      `json:"wall_time"`
	Entity       string         `json:"entity"`
	Event        string         `json:"event"`
	SimTime      int            `json:"sim_time"`
	WorkerID     int            `json:"worker_id,omitempty"`
	EventID      int            `json:"event_id,omitempty"`
	Target       int            `json:"target_worker,omitempty"`
	RollbackFrom int            `json:"rollback_from,omitempty"`
	RollbackTo   int            `json:"rollback_to,omitempty"`
	Details      map[string]any `json:"details,omitempty"`
}

func (cfg SimulationConfig) validate() error {
	if cfg.NumWorkers <= 0 {
		return errors.New("numWorkers must be > 0")
	}
	if cfg.TotalExternalEvents < cfg.NumWorkers {
		return errors.New("total events must be >= numWorkers")
	}
	if cfg.InternalMinEvents <= 0 || cfg.InternalMaxEvents < cfg.InternalMinEvents {
		return errors.New("invalid internal event count bounds")
	}
	if cfg.InternalMinJump <= 0 || cfg.InternalMaxJump < cfg.InternalMinJump {
		return errors.New("invalid internal jump bounds")
	}
	if cfg.ChannelBuffer <= 0 {
		return errors.New("channel buffer must be > 0")
	}
	if cfg.LogPath == "" {
		return errors.New("log path required")
	}
	return nil
}

func RunSimulation(cfg SimulationConfig) (*SimulationResult, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	logger, err := NewLogger(cfg.LogPath)
	if err != nil {
		return nil, err
	}
	defer logger.Close()
	start := time.Now()
	workerChans := make([]chan Evento, cfg.NumWorkers)
	workers := make([]*Worker, cfg.NumWorkers)
	var wg sync.WaitGroup
	for i := 0; i < cfg.NumWorkers; i++ {
		workerChans[i] = make(chan Evento, cfg.ChannelBuffer)
		workers[i] = NewWorker(i, cfg, workerChans[i], logger)
		wg.Add(1)
		go workers[i].Run(&wg)
	}
	scheduler := NewScheduler(cfg, workerChans, logger)
	scheduler.Run()
	for _, ch := range workerChans {
		close(ch)
	}
	wg.Wait()
	duration := time.Since(start)
	stats := make([]WorkerStats, len(workers))
	for i, worker := range workers {
		stats[i] = worker.stats
	}
	return &SimulationResult{
		Duration:         duration,
		EventsDispatched: cfg.TotalExternalEvents,
		WorkerStats:      stats,
	}, nil
}

func runSpeedupExperiment(base SimulationConfig) error {
	counts := []int{1, 2, 4, 8}
	results := make([]time.Duration, len(counts))
	for idx, workers := range counts {
		cfg := base
		cfg.NumWorkers = workers
		cfg.LogPath = fmt.Sprintf("speedup_w%d.log", workers)
		res, err := RunSimulation(cfg)
		if err != nil {
			return err
		}
		results[idx] = res.Duration
		fmt.Printf("Workers: %d \tDuration: %s \tLog: %s\n", workers, res.Duration, cfg.LogPath)
	}
	baseDuration := results[0]
	fmt.Println("\nSpeedup analysis (relative to 1 worker):")
	for idx, workers := range counts {
		speedup := float64(baseDuration) / float64(results[idx])
		fmt.Printf("Workers: %d \tSpeedup: %.2f\n", workers, speedup)
	}
	return nil
}

func main() {
	var (
		workers     = flag.Int("workers", 2, "Cantidad de workers en la simulación")
		events      = flag.Int("events", 40, "Cantidad total de eventos externos generados")
		logPath     = flag.String("log", "execution.log", "Archivo de log a generar")
		seed        = flag.Int64("seed", time.Now().UnixNano(), "Semilla utilizada para los generadores de eventos")
		channelBuf  = flag.Int("channel-buffer", 8, "Tamaño de los canales entre scheduler y workers")
		internalMin = flag.Int("internal-min", 1, "Cantidad mínima de eventos internos por evento externo")
		internalMax = flag.Int("internal-max", 3, "Cantidad máxima de eventos internos por evento externo")
		jumpMin     = flag.Int("jump-min", 1, "Avance mínimo del LVT causado por un evento interno")
		jumpMax     = flag.Int("jump-max", 5, "Avance máximo del LVT causado por un evento interno")
		speedup     = flag.Bool("speedup", false, "Ejecuta automáticamente la medición de speedup (1,2,4,8 workers)")
	)
	flag.Parse()
	cfg := SimulationConfig{
		NumWorkers:          *workers,
		TotalExternalEvents: *events,
		InternalMinEvents:   *internalMin,
		InternalMaxEvents:   *internalMax,
		InternalMinJump:     *jumpMin,
		InternalMaxJump:     *jumpMax,
		ChannelBuffer:       *channelBuf,
		LogPath:             *logPath,
		Seed:                *seed,
	}
	if *speedup {
		if err := runSpeedupExperiment(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "speedup experiment failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	res, err := RunSimulation(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "simulation failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Simulación completada en %s. Eventos despachados: %d\n", res.Duration, res.EventsDispatched)
	for _, s := range res.WorkerStats {
		fmt.Printf("Worker %d -> extern: %d, intern: %d, rollbacks: %d, checkpoints: %d, LVT final: %d\n",
			s.ID, s.ExternalEvents, s.InternalEvents, s.Rollbacks, s.CheckpointsBuilt, s.LastVirtualTime)
	}
	fmt.Printf("Logs guardados en %s\n", cfg.LogPath)
}
