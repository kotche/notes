package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"time"
)

var (
	// Метрика для общего количества отправленных заметок
	SentNotesGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sent_notes_total",
			Help: "Total number of notes sent to the notification bot",
		},
	)

	// Метрика для количества отправленных заметок по времени
	NotesSentCounterVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notes_sent_for_time",
			Help: "Number of notes sent, with timestamp as a label",
		},
		[]string{"timestamp"},
	)
)

func Init() {
	// Регистрируем метрики
	prometheus.MustRegister(SentNotesGauge)
	prometheus.MustRegister(NotesSentCounterVec)
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

func SendNotes(count int) {
	SentNotesGauge.Add(float64(count)) // Увеличиваем общую метрику

	// Увеличиваем счетчик с лейблом timestamp
	NotesSentCounterVec.WithLabelValues(time.Now().Format("2006-01-02 15:04:05")).Add(float64(count))
}
