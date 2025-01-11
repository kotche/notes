package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
)

var (
	// Объявляем метрику Counter (для rate показателей)
	NotesSentCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "notes_sent_total",
			Help: "Total number of notes sent",
		},
	)

	// Объявляем метрику Histogram (для response time показателей)
	ResponseTimeHistogram = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "response_time_seconds",
			Help:    "Response time in seconds",
			Buckets: prometheus.LinearBuckets(0.1, 0.1, 10), // Бакеты от 0.1 до 1.0 секунд
		},
	)
)

func Init() {
	// Регистрируем метрики
	prometheus.MustRegister(NotesSentCounter)
	prometheus.MustRegister(ResponseTimeHistogram)
}

func StartMetricsServer(port string) {
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Printf("metrics server running on %s", port)
		if err := http.ListenAndServe(port, nil); err != nil {
			log.Fatalf("failed to start metrics server: %v", err)
		}
	}()
}
