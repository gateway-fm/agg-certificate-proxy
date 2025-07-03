package metrics

import (
	"math/big"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Prometheus metrics
var (
	// Certificate metrics
	certificateTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "certificate_total",
			Help: "Total number of certificates open in the queue",
		},
		[]string{"status", "token"},
	)

	certificateValue = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "certificate_value",
			Help: "Value of certificates by token",
		},
		[]string{"token"},
	)
)

type Totals struct {
	GrandTotal     *big.Int
	FormattedTotal string
	CertCount      int
	Tokens         map[string]*big.Int
}

// ReportTotals updates Prometheus metrics with the provided totals
func ReportTotals(totals Totals) {
	// Update certificate count
	certificateTotal.WithLabelValues("open", "all").Add(float64(totals.CertCount))
}

// GetMetricsHandler returns an HTTP handler for Prometheus metrics
func WireUpHttpMetrics() {
	http.Handle("/metrics", promhttp.Handler())
}

// RegisterMetrics registers all metrics with the default registry
func RegisterMetrics() {
	// Metrics are automatically registered when created with promauto
	// This function can be used for any additional registration logic
}
