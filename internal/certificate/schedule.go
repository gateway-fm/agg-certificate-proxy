package certificate

import (
	"time"

	"github.com/go-co-op/gocron/v2"
	"log/slog"
)

// Scheduler handles periodic certificate processing.
type Scheduler struct {
	service   *Service
	scheduler gocron.Scheduler
}

// NewScheduler creates a new scheduler.
func NewScheduler(service *Service, interval time.Duration) (*Scheduler, error) {
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

// Stop halts the processing loop.
func (s *Scheduler) Stop() {
	slog.Info("stopping certificate scheduler...")
	if err := s.scheduler.Shutdown(); err != nil {
		slog.Error("error shutting down scheduler", "err", err)
	}
}

// processCertificates is the task that runs periodically.
func (s *Scheduler) processCertificates() {
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

	s.service.ProcessPendingCertificates()
}
