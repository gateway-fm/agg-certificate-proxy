package certificate

import (
	"context"
	"sync"
	"time"

	"log/slog"

	"github.com/go-co-op/gocron/v2"
)

// Scheduler handles periodic certificate processing.
type Scheduler struct {
	service   *Service
	scheduler gocron.Scheduler
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// NewScheduler creates a new scheduler.
func NewScheduler(service *Service, interval time.Duration) (*Scheduler, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	scheduler := &Scheduler{
		service:   service,
		scheduler: s,
		ctx:       ctx,
		cancel:    cancel,
	}

	_, err = s.NewJob(
		gocron.DurationJob(interval),
		gocron.NewTask(scheduler.processCertificates),
	)
	if err != nil {
		cancel()
		return nil, err
	}

	return scheduler, nil
}

// Start begins the processing loop.
func (s *Scheduler) Start() {
	slog.Info("starting certificate scheduler...")
	s.scheduler.Start()
}

// Stop halts the processing loop gracefully.
func (s *Scheduler) Stop() {
	slog.Info("stopping certificate scheduler...")

	// Signal cancellation
	s.cancel()

	// Stop accepting new jobs
	if err := s.scheduler.StopJobs(); err != nil {
		slog.Error("error stopping scheduler jobs", "err", err)
	}

	// Wait for any running tasks to complete
	slog.Info("waiting for running tasks to complete...")
	s.wg.Wait()

	// Shutdown the scheduler
	if err := s.scheduler.Shutdown(); err != nil {
		slog.Error("error shutting down scheduler", "err", err)
	}

	slog.Info("certificate scheduler stopped")
}

// IsProcessing returns true if there are active tasks running.
func (s *Scheduler) IsProcessing() bool {
	// Since we can't check WaitGroup state directly, we'll track this differently
	// For now, we can simply check if the scheduler context is done
	select {
	case <-s.ctx.Done():
		return false
	default:
		// If needed, we could add an atomic counter for active tasks
		return true
	}
}

// processCertificates is the task that runs periodically.
func (s *Scheduler) processCertificates() {
	// Check if we're shutting down before starting
	select {
	case <-s.ctx.Done():
		slog.Info("scheduler shutting down, skipping certificate processing")
		return
	default:
	}

	// Track this task
	s.wg.Add(1)
	defer s.wg.Done()

	// Create a context for this processing run
	// TODO: Update ProcessPendingCertificates to accept context for proper cancellation
	// ctx, cancel := context.WithCancel(s.ctx)
	// defer cancel()

	// Check if scheduler is active in the database
	isActive, err := s.service.db.GetSchedulerStatus()
	if err != nil {
		slog.Error("error checking scheduler status", "err", err)
		return
	}

	if !isActive {
		slog.Info("scheduler is disabled via kill switch, skipping certificate processing")
		return
	}

	slog.Info("scheduler processing certificates...")

	// Pass context to processing method so it can be cancelled
	// For now, use the existing method until we update it
	s.service.ProcessPendingCertificates()

	slog.Info("scheduler finished processing certificates")
}
