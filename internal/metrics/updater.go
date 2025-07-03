package metrics

import (
	"context"
	"log/slog"

	"github.com/gateway-fm/agg-certificate-proxy/internal/certificate"
)

type Updater struct {
	service *certificate.Service
	trigger chan struct{}
}

func NewUpdater(service *certificate.Service) *Updater {
	return &Updater{
		service: service,
		// buffered channel to avoid blocking and all we need to know is that "something"
		// has happened whilst we were busy
		trigger: make(chan struct{}, 1),
	}
}

func (u *Updater) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-u.trigger:
				u.UpdateMetrics()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (u *Updater) Trigger() {
	select {
	case u.trigger <- struct{}{}:
	default:
		// channel is full, so we don't need to do anything
	}
}

func (u *Updater) UpdateMetrics() {
	// do something
	slog.Info("updating metrics")
}
