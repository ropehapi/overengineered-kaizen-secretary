# OpenTelemetry + Jaeger — Design

**Spec**: `.specs/features/otel-jaeger/spec.md`
**Status**: Draft

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                  cmd/worker/main.go                     │
│                                                         │
│  telemetry.Init() ──────────────────────────────────┐  │
│                                                     │  │
│  c.AddFunc(expr, func() {                           │  │
│      ctx, span := tracer.Start(ctx, "rotina")       │  │
│      routines.RememberScoutMonthlyFees(ctx)         │  │
│      span.End()                                     │  │
│  })                                                 │  │
└─────────────────────────────────────────────────────┼──┘
                                                      │
                    OTLP HTTP (porta 4318)            │
                                                      ▼
┌──────────────────────────────────────────────────────────┐
│              internal/telemetry/tracer.go                │
│                                                          │
│  - Configura TracerProvider com OTLP HTTP exporter       │
│  - Resource: service.name = "kaizen-secretary"           │
│  - Retorna shutdown func para graceful cleanup           │
└──────────────────────────────────────────────────────────┘
                         │
                         │ OTLP/HTTP
                         ▼
              ┌─────────────────────┐
              │   Jaeger (docker)   │
              │   porta 4318 (OTLP) │
              │   porta 16686 (UI)  │
              └─────────────────────┘
```

**Fluxo de spans:**
```
Trace: RememberScoutMonthlyFees
  └── span: "routine.execute" (duração total da rotina)
        ├── span: "send_whatsapp_message" (recipient: +5511...)
        ├── span: "send_whatsapp_message" (recipient: +5521...)
        └── span: "send_whatsapp_message" (recipient: +5531...)
```

---

## Code Reuse Analysis

### Existing Components to Leverage

| Component | Location | How to Use |
|---|---|---|
| Logger init pattern | `internal/logger/logger.go` | Replicar o padrão `Init()` para `telemetry.Init()` |
| Routine function | `internal/routines/mensalidadesEscoteiro.go` | Adicionar `ctx context.Context` como parâmetro |
| HTTP client em rotinas | `mensalidadesEscoteiro.go` | Envolver com `otelhttp.NewTransport` |

### Integration Points

| Sistema | Método |
|---|---|
| `cron.AddFunc` | Wrapper que cria span raiz antes de chamar a rotina |
| `net/http` em rotinas | `otelhttp.NewTransport(http.DefaultTransport)` para auto-instrumentação |
| `docker-compose.yml` | Adicionar service `jaeger` com imagem oficial |

---

## Components

### `internal/telemetry/tracer.go`

- **Purpose**: Inicializar e configurar o TracerProvider global com exporter OTLP
- **Location**: `internal/telemetry/tracer.go`
- **Interfaces**:
  - `Init(ctx context.Context) (shutdown func(context.Context) error, err error)` — configura o provider global e retorna função de cleanup
- **Dependencies**: `go.opentelemetry.io/otel`, `go.opentelemetry.io/otel/exporters/otlp/otlphttp`
- **Reuses**: Padrão de `Init()` de `internal/logger/logger.go`

### Modificação em `cmd/worker/main.go`

- **Purpose**: Inicializar telemetria e envolver chamadas de rotina com span raiz
- **Location**: `cmd/worker/main.go` (modificação)
- **O que muda**:
  - Chamar `telemetry.Init()` após `logger.Init()`
  - Registrar `defer shutdown(ctx)` para graceful shutdown
  - Envolver cada `c.AddFunc` com criação de span raiz
- **Reuses**: Padrão existente de inicialização sequencial

### Modificação em `internal/routines/mensalidadesEscoteiro.go`

- **Purpose**: Adicionar spans filhos por envio de mensagem
- **Location**: `internal/routines/mensalidadesEscoteiro.go` (modificação)
- **O que muda**:
  - Assinatura passa a aceitar `ctx context.Context`
  - HTTP client usa `otelhttp.NewTransport`
  - Cada iteração cria span filho com atributo `recipient.phone`
- **Reuses**: Lógica de envio existente, apenas adiciona instrumentação

### `docker-compose.yml` — service Jaeger

- **Purpose**: Subir Jaeger All-in-One para receber e visualizar traces
- **Location**: `docker-compose.yml` (modificação)
- **Config**:
  ```yaml
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"  # UI
      - "4318:4318"    # OTLP HTTP
    environment:
      - COLLECTOR_OTLP_ENABLED=true
  ```

---

## Tech Decisions

| Decisão | Escolha | Rationale |
|---|---|---|
| Protocolo de exportação | OTLP HTTP (não gRPC) | Mais simples de configurar, sem dependência de gRPC |
| Jaeger image | `all-in-one` | Inclui collector + UI em uma imagem só — ideal para dev/demo |
| Propagação de contexto | `context.Context` passado explicitamente | Padrão Go idiomático, evita globals |
| HTTP instrumentation | `otelhttp.NewTransport` | Auto-instrumenta sem mudar código de negócio |

---

## Dependencies a adicionar

```
go.opentelemetry.io/otel
go.opentelemetry.io/otel/trace
go.opentelemetry.io/otel/sdk
go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp
go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp
```
