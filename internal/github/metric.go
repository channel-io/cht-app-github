package github

import (
	"fmt"

	"github.com/google/go-github/v60/github"
	"github.com/prometheus/client_golang/prometheus"
)

type ClientMetrics struct {
	requestCount *prometheus.CounterVec
	errorCount   *prometheus.CounterVec

	requestRateLimit     *prometheus.GaugeVec
	requestRateRemaining *prometheus.GaugeVec
}

const (
	labelInstallationID = "installation_id"
	labelOperation      = "operation"
)

func NewClientMetrics() *ClientMetrics {
	metrics := &ClientMetrics{
		requestCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "github",
				Name:      "requests_total",
				Help:      "Total number of requests made with github client",
			},
			[]string{labelInstallationID, labelOperation},
		),

		errorCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "github",
				Name:      "errors_total",
				Help:      "Total number of errors made with github client",
			},
			[]string{labelInstallationID, labelOperation},
		),

		requestRateLimit: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "github",
				Name:      "request_rate_limit",
				Help:      "The maximum number of requests that you can make per hour with github API",
			},
			[]string{labelInstallationID},
		),
		requestRateRemaining: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "github",
				Name:      "request_rate_remaining",
				Help:      "The number of requests remaining in the current rate limit window with github API",
			},
			[]string{labelInstallationID},
		),
	}

	return metrics
}

func (m *ClientMetrics) Describe(ch chan<- *prometheus.Desc) {
	m.requestCount.Describe(ch)
	m.errorCount.Describe(ch)
	m.requestRateLimit.Describe(ch)
	m.requestRateRemaining.Describe(ch)
}

func (m *ClientMetrics) Collect(ch chan<- prometheus.Metric) {
	m.requestCount.Collect(ch)
	m.errorCount.Collect(ch)
	m.requestRateLimit.Collect(ch)
	m.requestRateRemaining.Collect(ch)
}

func (m *ClientMetrics) onResponse(ctx InstallationContext, operation string, res *github.Response, err error) {
	installationID := fmt.Sprintf("%d", ctx.InstallationId)

	m.requestCount.
		WithLabelValues(installationID, operation).
		Inc()

	if err != nil {
		m.errorCount.
			WithLabelValues(installationID, operation).
			Inc()
		return
	}

	if res != nil {
		m.requestRateLimit.
			WithLabelValues(installationID).
			Set(float64(res.Rate.Limit))

		m.requestRateRemaining.
			WithLabelValues(installationID).
			Set(float64(res.Rate.Remaining))
	}
}
