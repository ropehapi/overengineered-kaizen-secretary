package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	RoutineExecutionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "routine_executions_total",
			Help: "Total de execuções de rotinas cron",
		},
		[]string{"routine"},
	)

	RoutineDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "routine_duration_seconds",
			Help:    "Duração das rotinas cron em segundos",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"routine"},
	)

	RoutineErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "routine_errors_total",
			Help: "Total de erros em rotinas cron",
		},
		[]string{"routine"},
	)

	MessagesSentTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messages_sent_total",
			Help: "Total de mensagens enviadas por rotina e status",
		},
		[]string{"routine", "status"},
	)
)

func Init() {
	prometheus.MustRegister(
		RoutineExecutionsTotal,
		RoutineDurationSeconds,
		RoutineErrorsTotal,
		MessagesSentTotal,
	)
}
