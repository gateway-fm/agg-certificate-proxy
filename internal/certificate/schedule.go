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
	wg        sync.WaitGroup
}

// NewScheduler creates a new scheduler with a parent context.
func NewScheduler(ctx context.Context, service *Service, interval time.Duration) (*Scheduler, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}

	scheduler := &Scheduler{
		service:   service,
		scheduler: s,
	}

	_, err = s.NewJob(
		gocron.DurationJob(interval),
		gocron.NewTask(scheduler.processCertificates),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
		gocron.WithContext(ctx),
	)
	if err != nil {
		return nil, err
	}

	return scheduler, nil
}

// Start begins the processing loop.
func (s *Scheduler) Start() {
	slog.Info("starting certificate scheduler...")
	s.scheduler.Start()
}

func (s *Scheduler) Stop() {
	slog.Info("stopping certificate scheduler...")

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

// processCertificates is the task that runs periodically - the context is passed by the gocron library
// which is assigned as part of the Scheduler constructor
func (s *Scheduler) processCertificates(ctx context.Context) {
	// Check if we're shutting down before starting
	select {
	case <-ctx.Done():
		slog.Info("scheduler shutting down, skipping certificate processing")
		return
	default:
	}

	// Track this task
	s.wg.Add(1)
	defer s.wg.Done()

	// Check if Scheduler is active in the database
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
	s.service.ProcessPendingCertificates(ctx)

	slog.Info("scheduler finished processing certificates")
}
