# Prometheus + Grafana — Design

**Spec**: `.specs/features/prometheus-grafana/spec.md`
**Status**: Draft

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                  cmd/worker/main.go                     │
│                                                         │
│  metrics.Init()  ──────────────────────────────────┐   │
│                                                    │   │
│  go http.ListenAndServe(":9090", promhttp.Handler())│   │
│                                                    │   │
│  c.AddFunc(expr, func() {                          │   │
│      routines.RememberScoutMonthlyFees()           │   │
│      // metrics coletadas dentro da rotina         │   │
│  })                                                │   │
└────────────────────────────────────────────────────┼───┘
                                                     │
                    HTTP scrape                      │
                    GET /metrics                     │
                                                     ▼
┌──────────────────────────────────────────────────────────┐
│              internal/metrics/metrics.go                 │
│                                                          │
│  var (                                                   │
│    RoutineExecutionsTotal *prometheus.CounterVec         │
│    RoutineDurationSeconds *prometheus.HistogramVec       │
│    RoutineErrorsTotal     *prometheus.CounterVec         │
│    MessagesSentTotal      *prometheus.CounterVec         │
│  )                                                       │
│                                                          │
│  func Init() { /* registra todas as métricas */ }        │
└──────────────────────────────────────────────────────────┘

              ┌─────────────┐         ┌──────────────┐
              │ Prometheus  │────────►│   Grafana    │
              │ porta 9091  │  query  │  porta 3001  │
              │ scrape :9090│         │  dashboard   │
              └─────────────┘         └──────────────┘
```

**Fluxo de coleta:**
```
Worker inicia
  └── metrics.Init() registra métricas no DefaultRegisterer
  └── goroutine: HTTP server em :9090 com promhttp.Handler()
  └── cron dispara rotina:
        ├── RoutineExecutionsTotal.Inc()
        ├── timer := prometheus.NewTimer(RoutineDurationSeconds)
        ├── per message: MessagesSentTotal.With(status="success/failure").Inc()
        └── timer.ObserveDuration()
```

---

## Code Reuse Analysis

### Existing Components to Leverage

| Component | Location | How to Use |
|---|---|---|
| Logger init pattern | `internal/logger/logger.go` | Replicar padrão `Init()` para `metrics.Init()` |
| Rotina existente | `internal/routines/mensalidadesEscoteiro.go` | Adicionar chamadas de métricas nos pontos de sucesso/falha |

### Integration Points

| Sistema | Método |
|---|---|
| `cmd/worker/main.go` | Chamar `metrics.Init()` e subir HTTP server em goroutine |
| Rotinas | Importar `internal/metrics` e chamar counters/histograms |
| `docker-compose.yml` | Adicionar services `prometheus` e `grafana` |
| `configs/` | Arquivos de configuração do Prometheus e Grafana |

---

## Components

### `internal/metrics/metrics.go`

- **Purpose**: Definir e registrar todas as métricas do worker como variáveis globais
- **Location**: `internal/metrics/metrics.go` (novo arquivo)
- **Interfaces**:
  - `Init()` — registra todas as métricas no `prometheus.DefaultRegisterer`
- **Métricas definidas**:
  ```go
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
          Help: "Total de mensagens enviadas",
      },
      []string{"routine", "status"},
  )
  ```
- **Dependencies**: `github.com/prometheus/client_golang/prometheus`

### Modificação em `cmd/worker/main.go`

- **Purpose**: Inicializar métricas e subir HTTP server para o endpoint `/metrics`
- **O que adicionar**:
  ```go
  metrics.Init()

  port := os.Getenv("METRICS_PORT")
  if port == "" { port = "9090" }

  go func() {
      http.Handle("/metrics", promhttp.Handler())
      if err := http.ListenAndServe(":"+port, nil); err != nil {
          slog.Error("metrics server failed", "error", err)
          os.Exit(1)
      }
  }()
  ```

### Modificação em `internal/routines/mensalidadesEscoteiro.go`

- **Purpose**: Registrar métricas nos pontos de execução da rotina
- **O que adicionar**:
  ```go
  // No início da função:
  metrics.RoutineExecutionsTotal.WithLabelValues("mensalidadesEscoteiro").Inc()
  timer := prometheus.NewTimer(metrics.RoutineDurationSeconds.WithLabelValues("mensalidadesEscoteiro"))
  defer timer.ObserveDuration()

  // Por mensagem com sucesso:
  metrics.MessagesSentTotal.WithLabelValues("mensalidadesEscoteiro", "success").Inc()

  // Por mensagem com falha:
  metrics.MessagesSentTotal.WithLabelValues("mensalidadesEscoteiro", "failure").Inc()
  ```

### `configs/prometheus.yml`

- **Purpose**: Configurar o Prometheus para scrape do worker
- **Location**: `configs/prometheus.yml` (novo arquivo)
- **Conteúdo**:
  ```yaml
  global:
    scrape_interval: 15s

  scrape_configs:
    - job_name: "kaizen-secretary"
      static_configs:
        - targets: ["app:9090"]
  ```

### `configs/grafana/provisioning/datasources/prometheus.yml`

- **Purpose**: Auto-provisionar datasource do Prometheus no Grafana
- **Location**: `configs/grafana/provisioning/datasources/prometheus.yml`

### `configs/grafana/provisioning/dashboards/`

- **Purpose**: Auto-provisionar dashboard básico do kaizen-secretary
- **Location**: `configs/grafana/provisioning/dashboards/`

### `docker-compose.yml` — services Prometheus + Grafana

```yaml
prometheus:
  image: prom/prometheus:latest
  ports:
    - "9091:9090"
  volumes:
    - ./configs/prometheus.yml:/etc/prometheus/prometheus.yml
  networks:
    - manda-pra-mim

grafana:
  image: grafana/grafana:latest
  ports:
    - "3001:3000"
  environment:
    - GF_SECURITY_ADMIN_PASSWORD=admin
  volumes:
    - ./configs/grafana/provisioning:/etc/grafana/provisioning
  networks:
    - manda-pra-mim
```

---

## Tech Decisions

| Decisão | Escolha | Rationale |
|---|---|---|
| Porta do metrics server | `9090` (default) | Porta padrão para exporters Prometheus |
| Porta do Prometheus no compose | `9091` | Evita conflito com o worker na `9090` |
| Porta do Grafana no compose | `3001` | Evita conflito com messaging-officer na `3000` |
| Métricas como vars globais | Sim (`internal/metrics`) | Padrão comum em Go para exporters Prometheus |
| Dashboard format | JSON provisioning | Reproduzível sem configuração manual no Grafana |

---

## Dependencies a adicionar

```
github.com/prometheus/client_golang/prometheus
github.com/prometheus/client_golang/prometheus/promhttp
```
