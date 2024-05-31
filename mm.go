package mm

import (
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type RegistrerGatherer interface {
	prometheus.Registerer
	prometheus.Gatherer
}

type MM struct {
	Namespace string
	Subsystem string
	Registry  RegistrerGatherer
}

func New(metricsRegistry RegistrerGatherer, namespace, subsystem string) *MM {
	if metricsRegistry == nil {
		metricsRegistry = prometheus.NewRegistry()
	}

	return &MM{
		Namespace: normalizeString(namespace),
		Subsystem: normalizeString(subsystem),
		Registry:  metricsRegistry,
	}
}

func (mm *MM) AddClientRequestsInFlight(next http.RoundTripper) http.RoundTripper {
	metric := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: mm.Namespace,
		Subsystem: mm.Subsystem,
		Name:      "http_client_in_flight_requests",
		Help:      "Total count of in-flight requests for the wrapped http client.",
	})

	mm.Registry.MustRegister(metric)

	return promhttp.InstrumentRoundTripperInFlight(metric, next)
}

func (mm *MM) AddClientRequestsCounter(next http.RoundTripper) http.RoundTripper {
	metric := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mm.Namespace,
			Subsystem: mm.Subsystem,
			Name:      "http_client_api_requests_total",
			Help:      "A counter for requests from the wrapped client.",
		},
		[]string{"code", "method"},
	)

	mm.Registry.MustRegister(metric)

	return promhttp.InstrumentRoundTripperCounter(metric, next)
}

func (mm *MM) AddClientTrace(next http.RoundTripper) http.RoundTripper {
	clientDNSLatencyVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: mm.Namespace,
			Subsystem: mm.Subsystem,
			Name:      "http_client_dns_duration_seconds",
			Help:      "Trace dns latency histogram.",
			Buckets:   []float64{.005, .01, .025, .05},
		},
		[]string{"event"},
	)

	clientTLSLatencyVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: mm.Namespace,
			Subsystem: mm.Subsystem,
			Name:      "http_client_tls_duration_seconds",
			Help:      "Trace tls latency histogram.",
			Buckets:   []float64{.05, .1, .25, .5},
		},
		[]string{"event"},
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

	mm.Registry.MustRegister(clientDNSLatencyVec, clientTLSLatencyVec)

	return promhttp.InstrumentRoundTripperTrace(clientTrace, next)
}

func (mm *MM) AddClientDuration(next http.RoundTripper) http.RoundTripper {
	metric := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: mm.Namespace,
			Subsystem: mm.Subsystem,
			Name:      "http_client_request_duration_seconds",
			Help:      "Trace http request latencies histogram.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{},
	)

	mm.Registry.MustRegister(metric)

	return promhttp.InstrumentRoundTripperDuration(metric, next)
}

func Build(base http.RoundTripper, middlewares ...func(http.RoundTripper) http.RoundTripper) http.RoundTripper {
	chain := base
	for _, middleware := range middlewares {
		chain = middleware(chain)
	}

	return chain
}

func (mm *MM) DefaultMiddlewares(baseTransport http.RoundTripper) http.RoundTripper {
	finalMiddleware := Build(
		baseTransport,
		mm.AddClientRequestsInFlight,
		mm.AddClientRequestsCounter,
		mm.AddClientTrace,
		mm.AddClientDuration,
	)
	return finalMiddleware
}

func normalizeString(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "-", "_")
	return s
}
