package meddleware

import (
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// CreateMiddleware creates the HTTP client metrics middleware, adds them to a Prometheus metric registry and returns a RoundTripper to be used by a HTTP client.
func Create(registry prometheus.Registerer, next http.RoundTripper, namespace, subsystem string) promhttp.RoundTripperFunc {
	if registry == nil {
		registry = prometheus.NewRegistry()
	}

	if next == nil {
		next = http.DefaultTransport
	}

	ns := normalizeString(namespace)
	ss := normalizeString(subsystem)

	clientInFlightGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: ns,
		Subsystem: ss,
		Name:      "http_client_in_flight_requests",
		Help:      "Total count of in-flight requests for the wrapped http client.",
	})

	clientAPIRequestsCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: ss,
			Name:      "client_api_requests_total",
			Help:      "A counter for requests from the wrapped client.",
		},
		[]string{"code", "method"},
	)

	clientDNSLatencyVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: ns,
			Subsystem: ss,
			Name:      "http_client_dns_duration_seconds",
			Help:      "Trace dns latency histogram.",
			Buckets:   []float64{.005, .01, .025, .05},
		},
		[]string{"event"},
	)

	clientTLSLatencyVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: ns,
			Subsystem: ss,
			Name:      "http_client_tls_duration_seconds",
			Help:      "Trace tls latency histogram.",
			Buckets:   []float64{.05, .1, .25, .5},
		},
		[]string{"event"},
	)

	clientHistVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: ns,
			Subsystem: ss,
			Name:      "http_client_request_duration_seconds",
			Help:      "Trace http request latencies histogram.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{},
	)

	clientTrace := &promhttp.InstrumentTrace{
		DNSStart: func(t float64) {
			clientDNSLatencyVec.WithLabelValues("dns_start").Observe(t)
		},
		DNSDone: func(t float64) {
			clientDNSLatencyVec.WithLabelValues("dns_done").Observe(t)
		},
		TLSHandshakeStart: func(t float64) {
			clientTLSLatencyVec.WithLabelValues("tls_handshake_start").Observe(t)
		},
		TLSHandshakeDone: func(t float64) {
			clientTLSLatencyVec.WithLabelValues("tls_handshake_done").Observe(t)
		},
	}

	registry.MustRegister(
		clientInFlightGauge,
		clientAPIRequestsCounter,
		clientDNSLatencyVec,
		clientTLSLatencyVec,
		clientHistVec,
	)

	return promhttp.InstrumentRoundTripperInFlight(clientInFlightGauge,
		promhttp.InstrumentRoundTripperCounter(clientAPIRequestsCounter,
			promhttp.InstrumentRoundTripperTrace(clientTrace,
				promhttp.InstrumentRoundTripperDuration(clientHistVec, next),
			),
		),
	)
}

func normalizeString(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "-", "_")
	return s
}
