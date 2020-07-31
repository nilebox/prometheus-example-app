package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

var (
	logger, _ = zap.NewProduction()
	reg          = prometheus.NewRegistry()
	requestCount = promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests by status code and method.",
		},
		[]string{"code", "method"},
	)
	histogram = promauto.With(reg).NewHistogram(prometheus.HistogramOpts{
		Name:    "random_numbers",
		Help:    "A histogram of normally distributed random numbers.",
		Buckets: prometheus.LinearBuckets(-3, .1, 61),
	})
)

// Generates random data for a histogram
func Random() {
	for {
		histogram.Observe(rand.NormFloat64())
	}
}

// PollItself polls the HTTP endpoint to generate synthetic "traffic"
func PollItself() {
	for {
		resp, err := http.Get("http://localhost:1234/")
		if err != nil {
			logger.Sugar().Errorf("HTTP request failed: %w", err)
		} else {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				logger.Sugar().Errorf("Failed to read response: %w", err)
			} else {
				logger.Sugar().Infof("Response: %s", string(body))
				_ = resp.Body.Close()
			}
		}
		time.Sleep(time.Second * time.Duration(rand.Intn(10)))
	}
}

func main() {
	go Random()
	go PollItself()

	// Example HTTP handler
	http.Handle("/", promhttp.InstrumentHandlerCounter(
		requestCount,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprint(w, "Hello, world!")
		}),
	))
	// Expose Prometheus metrics
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	logger.Info("Starting HTTP server")
	err := http.ListenAndServe(":1234", nil)

	if err != nil {
		logger.Sugar().Errorf("ListenAndServe failed: %w", err)
	}
}