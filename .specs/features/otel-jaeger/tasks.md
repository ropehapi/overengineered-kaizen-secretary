# OpenTelemetry + Jaeger — Tasks

**Design**: `.specs/features/otel-jaeger/design.md`
**Status**: Draft

---

## Execution Plan

```
Phase 1 (Sequential — Foundation):
  T1 → T2

Phase 2 (Parallel — Instrumentação):
  T2 complete, então:
    ├── T3 [P]  (modificar rotina)
    └── T4 [P]  (adicionar Jaeger ao docker-compose)

Phase 3 (Sequential — Integração):
  T3 + T4 complete, então:
    T5 → T6
```

---

## Task Breakdown

### T1: Adicionar dependências OTel ao go.mod

**What**: Rodar `go get` para adicionar os pacotes OpenTelemetry necessários ao projeto.
**Where**: `go.mod` / `go.sum`
**Depends on**: Nenhuma
**Requirement**: OTEL-01

**Pacotes a adicionar**:
```
go.opentelemetry.io/otel
go.opentelemetry.io/otel/trace
go.opentelemetry.io/otel/sdk
go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp
go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp
```

**Done when**:
- [ ] `go.mod` contém todos os pacotes listados
- [ ] `go build ./...` compila sem erros após adicionar as dependências

**Verify**:
```bash
go build ./...
```

**Commit**: `chore: add opentelemetry dependencies`

---

### T2: Criar `internal/telemetry/tracer.go`

**What**: Implementar a função `Init()` que configura o TracerProvider global com exporter OTLP HTTP.
**Where**: `internal/telemetry/tracer.go` (novo arquivo)
**Depends on**: T1
**Reuses**: Padrão de `Init()` de `internal/logger/logger.go`
**Requirement**: OTEL-01

**Implementação esperada**:
```go
package telemetry

import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
    "os"
)

func Init(ctx context.Context) (func(context.Context) error, error) {
    endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
    if endpoint == "" {
        endpoint = "http://localhost:4318"
    }

    exporter, err := otlptracehttp.New(ctx,
        otlptracehttp.WithEndpoint(endpoint),
        otlptracehttp.WithInsecure(),
    )
    // ... configura TracerProvider, seta como global, retorna shutdown
}
```

**Done when**:
- [ ] Arquivo `internal/telemetry/tracer.go` criado
- [ ] Função `Init(ctx) (shutdown, error)` exportada
- [ ] Resource `service.name = "kaizen-secretary"` configurado
- [ ] `go build ./...` compila sem erros

**Verify**:
```bash
go build ./...
go vet ./internal/telemetry/...
```

**Commit**: `feat(otel): create tracer initialization package`

---

### T3: Instrumentar `cmd/worker/main.go` com tracer e spans raiz [P]

**What**: Inicializar o tracer no startup e envolver cada rotina cron com um span raiz.
**Where**: `cmd/worker/main.go` (modificação)
**Depends on**: T2
**Reuses**: Padrão de inicialização sequencial existente (após `logger.Init()`)
**Requirement**: OTEL-01, OTEL-02

**O que adicionar**:
```go
// Após logger.Init():
ctx := context.Background()
shutdown, err := telemetry.Init(ctx)
if err != nil {
    slog.Error("failed to initialize telemetry", "error", err)
    os.Exit(1)
}
defer shutdown(ctx)

// Envolver AddFunc:
c.AddFunc(expr, func() {
    ctx, span := otel.Tracer("kaizen-secretary").Start(context.Background(), "RememberScoutMonthlyFees")
    defer span.End()
    routines.RememberScoutMonthlyFees(ctx)
})
```

**Done when**:
- [ ] `telemetry.Init()` chamado após `logger.Init()` no main
- [ ] `defer shutdown(ctx)` registrado para graceful shutdown
- [ ] Cada `c.AddFunc` envolve a rotina em um span com nome da rotina
- [ ] Assinatura de `RememberScoutMonthlyFees` atualizada para aceitar `ctx context.Context`
- [ ] `go build ./cmd/worker` compila sem erros

**Verify**:
```bash
go build ./cmd/worker
```

**Commit**: `feat(otel): initialize tracer in main and wrap routines with root spans`

---

### T4: Adicionar Jaeger ao `docker-compose.yml` [P]

**What**: Adicionar o service `jaeger` com imagem `all-in-one` e portas UI + OTLP expostas.
**Where**: `docker-compose.yml` (modificação)
**Depends on**: T2
**Requirement**: OTEL-04

**Config a adicionar**:
```yaml
jaeger:
  image: jaegertracing/all-in-one:latest
  container_name: kaizen-jaeger
  ports:
    - "16686:16686"   # UI
    - "4318:4318"     # OTLP HTTP receiver
  environment:
    - COLLECTOR_OTLP_ENABLED=true
  networks:
    - manda-pra-mim
```

**Variável de ambiente a adicionar no service `app`**:
```yaml
environment:
  - OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318
```

**Done when**:
- [ ] Service `jaeger` adicionado ao `docker-compose.yml`
- [ ] Porta `16686` (UI) e `4318` (OTLP) expostas
- [ ] Service `app` tem `OTEL_EXPORTER_OTLP_ENDPOINT` apontando para `jaeger`
- [ ] `docker compose config` valida sem erros

**Verify**:
```bash
docker compose config
docker compose up jaeger -d
curl -s http://localhost:16686 | grep -i jaeger
```

**Commit**: `feat(otel): add Jaeger all-in-one to docker-compose`

---

### T5: Instrumentar `RememberScoutMonthlyFees` com spans filhos

**What**: Adicionar span filho por iteração de destinatário na rotina, com atributos de `recipient.phone` e status HTTP.
**Where**: `internal/routines/mensalidadesEscoteiro.go` (modificação)
**Depends on**: T3 (assinatura com ctx já atualizada)
**Requirement**: OTEL-03

**O que adicionar por iteração**:
```go
func RememberScoutMonthlyFees(ctx context.Context) {
    tracer := otel.Tracer("kaizen-secretary")

    for name, phone := range contributors {
        _, span := tracer.Start(ctx, "send_whatsapp_message")
        span.SetAttributes(
            attribute.String("recipient.phone", phone),
            attribute.String("recipient.name", name),
        )

        // ... lógica de envio existente ...

        if err != nil {
            span.RecordError(err)
            span.SetStatus(codes.Error, err.Error())
        } else {
            span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
        }
        span.End()
    }
}
```

**Done when**:
- [ ] Função aceita `ctx context.Context` como primeiro parâmetro
- [ ] Cada iteração cria e encerra um span filho
- [ ] Span tem atributo `recipient.phone`
- [ ] Erros HTTP são registrados com `span.RecordError` e `span.SetStatus(codes.Error, ...)`
- [ ] `go build ./...` compila sem erros

**Verify**:
```bash
go build ./...
go vet ./internal/routines/...
```

**Commit**: `feat(otel): add child spans per message in scout monthly fees routine`

---

### T6: Validação end-to-end — traces no Jaeger UI

**What**: Alterar temporariamente o cron para disparar em ~10 segundos, subir o stack e validar os traces no Jaeger UI.
**Where**: `cmd/worker/main.go` (modificação temporária de teste)
**Depends on**: T3, T4, T5
**Requirement**: OTEL-01, OTEL-02, OTEL-03, OTEL-04

**Passos de validação**:
1. Alterar cron para `"*/10 * * * * *"` (a cada 10 segundos)
2. `docker compose up --build`
3. Aguardar 10-15 segundos
4. Abrir `http://localhost:16686`
5. Selecionar service `kaizen-secretary`
6. Verificar trace com spans filhos
7. Reverter o cron para a expressão correta

**Done when**:
- [ ] Jaeger UI (`localhost:16686`) exibe service `kaizen-secretary`
- [ ] Trace `RememberScoutMonthlyFees` aparece com spans filhos
- [ ] Cada span filho tem atributo `recipient.phone`
- [ ] Cron revertido para expressão original após validação
- [ ] `go build ./...` limpo

**Commit**: `test(otel): validate end-to-end traces in Jaeger UI`

---

## Parallel Execution Map

```
Phase 1 (Sequential):
  T1 ──→ T2

Phase 2 (Parallel):
  T2 complete, então:
    ├── T3 [P]
    └── T4 [P]

Phase 3 (Sequential):
  T3 + T4 completos, então:
    T5 ──→ T6
```

---

## Task Granularity Check

| Task | Escopo | Status |
|---|---|---|
| T1: Adicionar dependências | `go.mod` | ✅ Granular |
| T2: Criar tracer.go | 1 arquivo novo | ✅ Granular |
| T3: Instrumentar main.go | 1 arquivo, inicialização | ✅ Granular |
| T4: Adicionar Jaeger ao compose | 1 arquivo, 1 service | ✅ Granular |
| T5: Spans filhos na rotina | 1 arquivo, instrumentação | ✅ Granular |
| T6: Validação E2E | Teste manual | ✅ Granular |
