package certificate

import (
	"log"
	"time"

	"github.com/go-co-op/gocron/v2"
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
	log.Println("Starting certificate scheduler...")
	s.scheduler.Start()
}

// Stop halts the processing loop.
func (s *Scheduler) Stop() {
	log.Println("Stopping certificate scheduler...")
	if err := s.scheduler.Shutdown(); err != nil {
		log.Printf("error shutting down scheduler: %v", err)
	}
}

// processCertificates is the task that runs periodically.
func (s *Scheduler) processCertificates() {
	s.service.ProcessPendingCertificates()
}
