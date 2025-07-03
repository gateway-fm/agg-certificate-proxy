package metrics

import (
	"fmt"
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
			Name: "certificate_total_count",
			Help: "Total number of certificates open in the queue",
		},
		[]string{},
	)
	certificateTotalEth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "certificate_total_eth",
			Help: "Total value of ETH open in the queue across all networks",
		},
		[]string{},
	)
)

type Totals struct {
	GrandTotal *big.Int
	CertCount  int
	Networks   map[uint32]*big.Int
	Tokens     map[string]*big.Int
}

type PrometheusReporter struct {
	networkGaugeVec map[uint32]*prometheus.GaugeVec
}

func NewPrometheusReporter(networks []uint32) *PrometheusReporter {
	gauges := make(map[uint32]*prometheus.GaugeVec)
	for _, network := range networks {
		gauges[network] = promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: fmt.Sprintf("network_%d_total_eth", network),
				Help: fmt.Sprintf("Total amount of tokens bridged on network %d in ETH", network),
			},
			[]string{},
		)
	}

	return &PrometheusReporter{
		networkGaugeVec: gauges,
	}
}

// ReportTotals updates Prometheus metrics with the provided totals
func (r *PrometheusReporter) ReportTotals(totals Totals) {
	// Update certificate count
	certificateTotal.WithLabelValues().Add(float64(totals.CertCount))

	grandTotal := big.NewInt(0)

	for network, vec := range r.networkGaugeVec {
		if total, ok := totals.Networks[network]; ok {
			vec.WithLabelValues().Set(weiToEth(total))
			grandTotal.Add(grandTotal, total)
		} else {
			vec.WithLabelValues().Set(0)
		}
	}

	certificateTotalEth.WithLabelValues().Set(weiToEth(grandTotal))
}

// GetMetricsHandler returns an HTTP handler for Prometheus metrics
func (r *PrometheusReporter) WireUpHttpMetrics() {
	http.Handle("/metrics", promhttp.Handler())
}

func weiToEth(wei *big.Int) float64 {
	weiPerEth := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	ethFloat := new(big.Float).Quo(new(big.Float).SetInt(wei), new(big.Float).SetInt(weiPerEth))
	ethFloat.SetPrec(64)
	floatTotal, _ := ethFloat.Float64()
	return floatTotal
}
