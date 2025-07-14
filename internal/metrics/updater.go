package metrics

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/big"
	"strconv"
	"time"

	"github.com/gateway-fm/agg-certificate-proxy/internal/certificate"
)

type Updater struct {
	service  *certificate.Service
	trigger  chan struct{}
	reporter *PrometheusReporter
}

func NewUpdater(service *certificate.Service, reporter *PrometheusReporter) *Updater {
	return &Updater{
		service:  service,
		reporter: reporter,
		// buffered channel to avoid blocking and all we need to know is that "something"
		// has happened whilst we were busy
		trigger: make(chan struct{}, 256),
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

	// another routine to update based on a trigger.  We need this
	// because if a few override requests land or a number of certificates
	// there is a chance the metrics won't be updated to reflect the new state
	// because we use the buffered channel to ensure there isn't overwork done
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				slog.Info("timer based updating of metrics")
				u.trigger <- struct{}{}
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
	unprocessed, err := u.service.GetUnprocessedCertificates()
	if err != nil {
		slog.Error("failed to get unprocessed certificates", "err", err)
		return
	}

	totals := Totals{
		CertCount: len(unprocessed),
	}

	networks := make(map[uint32]*big.Int)

	for _, cert := range unprocessed {
		if cert.Metadata == "" {
			continue
		}

		var meta map[string]interface{}
		if err := json.Unmarshal([]byte(cert.Metadata), &meta); err == nil {
			network := meta["network_id"].(float64)
			if _, ok := networks[uint32(network)]; !ok {
				networks[uint32(network)] = big.NewInt(0)
			}

			if bridgeExits, ok := meta["bridge_exits"].([]interface{}); ok {
				for _, exit := range bridgeExits {
					if beMap, ok := exit.(map[string]interface{}); ok {
						if amountStr, ok := beMap["amount"].(string); ok {
							if amount, err := strconv.ParseUint(amountStr, 10, 64); err == nil {
								asBig := big.NewInt(0).SetUint64(amount)
								tot := networks[uint32(network)]
								tot.Add(tot, asBig)
							}
						}
					}
				}
			}
		}
	}

	totals.Networks = networks

	u.reporter.ReportTotals(totals)
}
