# Prometheus + Grafana — Tasks

**Design**: `.specs/features/prometheus-grafana/design.md`
**Status**: Draft

---

## Execution Plan

```
Phase 1 (Sequential — Foundation):
  T1 → T2

Phase 2 (Parallel — Core):
  T2 complete, então:
    ├── T3 [P]  (servidor HTTP de métricas no main)
    ├── T4 [P]  (métricas na rotina)
    └── T5 [P]  (configs Prometheus + Grafana)

Phase 3 (Sequential — Integração):
  T3 + T4 + T5 completos, então:
    T6 → T7
```

---

## Task Breakdown

### T1: Adicionar dependência `client_golang` ao go.mod

**What**: Rodar `go get` para adicionar o client Prometheus ao projeto.
**Where**: `go.mod` / `go.sum`
**Depends on**: Nenhuma
**Requirement**: PROM-01

**Pacote a adicionar**:
```
github.com/prometheus/client_golang
```

**Done when**:
- [ ] `go.mod` contém `github.com/prometheus/client_golang`
- [ ] `go build ./...` compila sem erros

**Verify**:
```bash
go build ./...
```

**Commit**: `chore: add prometheus client_golang dependency`

---

### T2: Criar `internal/metrics/metrics.go`

**What**: Definir e registrar todas as métricas do worker como variáveis globais com função `Init()`.
**Where**: `internal/metrics/metrics.go` (novo arquivo)
**Depends on**: T1
**Reuses**: Padrão de `Init()` de `internal/logger/logger.go`
**Requirement**: PROM-01, PROM-02, PROM-03

**Métricas a definir**:
- `routine_executions_total` (CounterVec, label: `routine`)
- `routine_duration_seconds` (HistogramVec, label: `routine`, buckets padrão)
- `routine_errors_total` (CounterVec, label: `routine`)
- `messages_sent_total` (CounterVec, labels: `routine`, `status`)

**Done when**:
- [ ] Arquivo `internal/metrics/metrics.go` criado
- [ ] As 4 métricas definidas como variáveis exportadas
- [ ] Função `Init()` registra todas no `prometheus.DefaultRegisterer`
- [ ] `go build ./...` compila sem erros

**Verify**:
```bash
go build ./...
go vet ./internal/metrics/...
```

**Commit**: `feat(metrics): create metrics package with routine and message counters`

---

### T3: Subir servidor HTTP de métricas em `cmd/worker/main.go` [P]

**What**: Inicializar métricas e subir goroutine com HTTP server expondo `/metrics`.
**Where**: `cmd/worker/main.go` (modificação)
**Depends on**: T2
**Requirement**: PROM-01

**O que adicionar** (após `logger.Init()`):
```go
metrics.Init()

port := os.Getenv("METRICS_PORT")
if port == "" {
    port = "9090"
}
go func() {
    http.Handle("/metrics", promhttp.Handler())
    if err := http.ListenAndServe(":"+port, nil); err != nil {
        slog.Error("metrics server failed", "error", err)
        os.Exit(1)
    }
}()
```

**Done when**:
- [ ] `metrics.Init()` chamado no startup
- [ ] Goroutine com HTTP server em `:9090` (ou `METRICS_PORT`) iniciada
- [ ] `curl http://localhost:9090/metrics` retorna métricas no formato texto
- [ ] `go build ./cmd/worker` compila sem erros

**Verify**:
```bash
go build ./cmd/worker
go run cmd/worker/main.go &
sleep 2
curl -s http://localhost:9090/metrics | grep "# HELP"
kill %1
```

**Commit**: `feat(metrics): expose /metrics HTTP endpoint in worker`

---

### T4: Instrumentar `RememberScoutMonthlyFees` com métricas [P]

**What**: Adicionar chamadas de métricas na rotina nos pontos de execução, sucesso e falha de mensagens.
**Where**: `internal/routines/mensalidadesEscoteiro.go` (modificação)
**Depends on**: T2
**Requirement**: PROM-02, PROM-03

**O que adicionar**:
```go
import (
    "github.com/ropehapi/kaizen-secretary/internal/metrics"
    "github.com/prometheus/client_golang/prometheus"
)

func RememberScoutMonthlyFees() {
    const routineName = "mensalidadesEscoteiro"

    metrics.RoutineExecutionsTotal.WithLabelValues(routineName).Inc()
    timer := prometheus.NewTimer(
        metrics.RoutineDurationSeconds.WithLabelValues(routineName),
    )
    defer timer.ObserveDuration()

    // ... lógica existente ...

    // por mensagem com sucesso:
    metrics.MessagesSentTotal.WithLabelValues(routineName, "success").Inc()

    // por mensagem com falha:
    metrics.MessagesSentTotal.WithLabelValues(routineName, "failure").Inc()
}
```

**Done when**:
- [ ] `RoutineExecutionsTotal` incrementado ao início da rotina
- [ ] `RoutineDurationSeconds` observado via `prometheus.NewTimer` + `defer ObserveDuration()`
- [ ] `MessagesSentTotal` com label `"success"` incrementado por envio bem-sucedido
- [ ] `MessagesSentTotal` com label `"failure"` incrementado por envio falho
- [ ] `go build ./...` compila sem erros

**Verify**:
```bash
go build ./...
go vet ./internal/routines/...
```

**Commit**: `feat(metrics): instrument scout monthly fees routine with prometheus metrics`

---

### T5: Criar configs de Prometheus e Grafana [P]

**What**: Criar os arquivos de configuração para Prometheus scrape e Grafana provisioning (datasource + dashboard).
**Where**: `configs/` (novo diretório)
**Depends on**: T2
**Requirement**: PROM-04

**Arquivos a criar**:

**`configs/prometheus.yml`**:
```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: "kaizen-secretary"
    static_configs:
      - targets: ["app:9090"]
```

**`configs/grafana/provisioning/datasources/prometheus.yml`**:
```yaml
apiVersion: 1
datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
```

**`configs/grafana/provisioning/dashboards/dashboard.yml`**:
```yaml
apiVersion: 1
providers:
  - name: kaizen-secretary
    folder: Kaizen
    type: file
    options:
      path: /etc/grafana/provisioning/dashboards
```

**`configs/grafana/provisioning/dashboards/kaizen-secretary.json`**:
Dashboard JSON com panels:
- Panel 1: `routine_executions_total` — total de execuções (stat)
- Panel 2: `rate(messages_sent_total[5m])` — taxa de envio por status (time series)
- Panel 3: `routine_duration_seconds` — duração média (gauge)
- Panel 4: `messages_sent_total` por status — pie chart

**Done when**:
- [ ] `configs/prometheus.yml` criado com scrape do `app:9090`
- [ ] `configs/grafana/provisioning/datasources/prometheus.yml` criado
- [ ] `configs/grafana/provisioning/dashboards/dashboard.yml` criado
- [ ] `configs/grafana/provisioning/dashboards/kaizen-secretary.json` criado com os 4 panels
- [ ] Todos os arquivos são YAML/JSON válidos

**Verify**:
```bash
python3 -c "import yaml; yaml.safe_load(open('configs/prometheus.yml'))" && echo "prometheus.yml OK"
python3 -c "import json; json.load(open('configs/grafana/provisioning/dashboards/kaizen-secretary.json'))" && echo "dashboard.json OK"
```

**Commit**: `feat(metrics): add Prometheus and Grafana provisioning configs`

---

### T6: Adicionar Prometheus e Grafana ao `docker-compose.yml`

**What**: Adicionar os services `prometheus` e `grafana` ao docker-compose com volumes apontando para os configs criados no T5.
**Where**: `docker-compose.yml` (modificação)
**Depends on**: T3, T5
**Requirement**: PROM-04

**Services a adicionar**:
```yaml
prometheus:
  image: prom/prometheus:latest
  container_name: kaizen-prometheus
  ports:
    - "9091:9090"
  volumes:
    - ./configs/prometheus.yml:/etc/prometheus/prometheus.yml:ro
  networks:
    - manda-pra-mim
  depends_on:
    - app

grafana:
  image: grafana/grafana:latest
  container_name: kaizen-grafana
  ports:
    - "3001:3000"
  environment:
    - GF_SECURITY_ADMIN_PASSWORD=admin
    - GF_USERS_ALLOW_SIGN_UP=false
  volumes:
    - ./configs/grafana/provisioning:/etc/grafana/provisioning:ro
  networks:
    - manda-pra-mim
  depends_on:
    - prometheus
```

**Variável de ambiente a adicionar no service `app`**:
```yaml
environment:
  - METRICS_PORT=9090
```

**Done when**:
- [ ] Services `prometheus` e `grafana` adicionados
- [ ] Volumes dos configs mapeados corretamente
- [ ] `docker compose config` valida sem erros
- [ ] Prometheus disponível em `localhost:9091`
- [ ] Grafana disponível em `localhost:3001`

**Verify**:
```bash
docker compose config
docker compose up prometheus grafana -d
sleep 5
curl -s http://localhost:9091/-/healthy && echo "Prometheus OK"
curl -s http://localhost:3001/api/health | grep ok && echo "Grafana OK"
docker compose down
```

**Commit**: `feat(metrics): add Prometheus and Grafana to docker-compose`

---

### T7: Validação end-to-end — métricas no Grafana

**What**: Subir o stack completo, acionar a rotina, e validar que as métricas aparecem no Grafana.
**Where**: `cmd/worker/main.go` (modificação temporária para acionar rotina)
**Depends on**: T4, T6
**Requirement**: PROM-01, PROM-02, PROM-03, PROM-04

**Passos de validação**:
1. `docker compose up --build`
2. Verificar `curl http://localhost:9090/metrics | grep routine_executions`
3. Alterar cron para disparar em 10s (temporário) e aguardar execução
4. Verificar `curl http://localhost:9090/metrics | grep messages_sent_total`
5. Abrir `http://localhost:9091/targets` — target `kaizen-secretary` deve estar `UP`
6. Abrir `http://localhost:3001` (admin/admin) — dashboard "Kaizen Secretary" deve estar disponível
7. Reverter cron para expressão original

**Done when**:
- [ ] `curl localhost:9090/metrics` retorna as 4 métricas definidas
- [ ] Prometheus `localhost:9091/targets` mostra `kaizen-secretary` como `UP`
- [ ] Grafana `localhost:3001` tem datasource Prometheus funcionando
- [ ] Dashboard "Kaizen Secretary" carrega com panels (podem estar vazios se rotina não rodou)
- [ ] Cron revertido para expressão original
- [ ] `go build ./...` limpo

**Commit**: `test(metrics): validate end-to-end metrics pipeline with Prometheus and Grafana`

---

## Parallel Execution Map

```
Phase 1 (Sequential):
  T1 ──→ T2

Phase 2 (Parallel):
  T2 complete, então:
    ├── T3 [P]
    ├── T4 [P]
    └── T5 [P]

Phase 3 (Sequential):
  T3 + T4 + T5 completos, então:
    T6 ──→ T7
```

---

## Task Granularity Check

| Task | Escopo | Status |
|---|---|---|
| T1: Adicionar dependência | `go.mod` | ✅ Granular |
| T2: Criar metrics.go | 1 arquivo novo | ✅ Granular |
| T3: HTTP server no main | 1 modificação focada | ✅ Granular |
| T4: Instrumentar rotina | 1 arquivo, pontos de métrica | ✅ Granular |
| T5: Configs Prometheus/Grafana | Arquivos de config | ✅ Granular |
| T6: Compose Prometheus+Grafana | 1 arquivo, 2 services | ✅ Granular |
| T7: Validação E2E | Teste manual | ✅ Granular |
