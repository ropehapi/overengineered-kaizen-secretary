# RiverQueue — Design

**Spec**: `.specs/features/river-queue/spec.md`
**Status**: Draft

---

## Architecture Overview

```
ANTES:
  cron → RememberScoutMonthlyFees() → HTTP → messaging-officer

DEPOIS:
  cron → InsertJobs(N destinatários) ──→ [PostgreSQL: river_jobs]
                                                    │
                                         River Worker (goroutine)
                                                    │
                                         SendWhatsAppMessageWorker.Work()
                                                    │
                                              HTTP POST
                                                    │
                                          messaging-officer
```

**Componentes no mesmo processo** (`cmd/worker/main.go`):
- `robfig/cron` — continua agendando (não é substituído)
- `river.Client` — insere jobs no banco
- `river.Workers` — processa jobs (goroutine separada, mesmo processo)

```
cmd/worker/main.go
  ├── logger.Init()
  ├── db = pgxpool.New(DATABASE_URL)         ← conexão ao PostgreSQL
  ├── riverqueue.Init(db)                    ← cria client + workers
  │     ├── river.AddWorker(workers, &SendWhatsAppMessageWorker{})
  │     └── riverClient, riverWorkers = ...
  │
  ├── c.AddFunc(expr, func() {
  │       jobs := buildJobs(contributors)    ← N InsertManyParams
  │       riverClient.InsertMany(ctx, jobs)  ← fan-out atômico
  │   })
  │
  ├── riverWorkers.Start(ctx)                ← inicia processamento
  └── select {}
```

---

## Code Reuse Analysis

### Existing Components to Leverage

| Component | Location | How to Use |
|---|---|---|
| Lógica de mensagem | `internal/routines/mensalidadesEscoteiro.go` | Extrair `buildMessage(name, phone)` para reuso no worker |
| HTTP client / envio | `mensalidadesEscoteiro.go` | Mover lógica de POST para `SendWhatsAppMessageWorker.Work()` |
| Map de contribuintes | `mensalidadesEscoteiro.go` | Exportar como função pública `Contributors()` |

### Integration Points

| Sistema | Método |
|---|---|
| PostgreSQL | `pgxpool.New` — pool de conexões, usado pelo River internamente |
| `docker-compose.yml` | Adicionar service `postgres` |
| `cmd/worker/main.go` | Integrar River client + workers no startup |

---

## Components

### `internal/riverqueue/setup.go`

- **Purpose**: Inicializar o River client e workers, criar schema no banco
- **Location**: `internal/riverqueue/setup.go` (novo arquivo)
- **Interfaces**:
  - `Init(ctx, pool) (*river.Client[pgx.Tx], *river.Workers, error)` — retorna client para inserção e workers para processamento
- **O que faz**:
  - Registra `SendWhatsAppMessageWorker` nos workers
  - Cria `river.Client` com o pool pgx
  - Roda migration do schema River (`rivermigrate`)
  - Retorna client e workers para uso no `main.go`

### `internal/riverqueue/jobs.go`

- **Purpose**: Definir o tipo de job e seus args
- **Location**: `internal/riverqueue/jobs.go` (novo arquivo)
- **Tipos**:
  ```go
  type SendWhatsAppMessageArgs struct {
      RecipientPhone string `json:"recipient_phone"`
      RecipientName  string `json:"recipient_name"`
      Message        string `json:"message"`
  }

  func (SendWhatsAppMessageArgs) Kind() string {
      return "send_whatsapp_message"
  }
  ```

### `internal/riverqueue/worker.go`

- **Purpose**: Implementar o worker que processa um job de envio de mensagem
- **Location**: `internal/riverqueue/worker.go` (novo arquivo)
- **Tipos**:
  ```go
  type SendWhatsAppMessageWorker struct {
      river.WorkerDefaults[SendWhatsAppMessageArgs]
  }

  func (w *SendWhatsAppMessageWorker) Work(ctx context.Context, job *river.Job[SendWhatsAppMessageArgs]) error {
      // Chama messaging-officer com job.Args.RecipientPhone + Message
      // Retorna erro para acionar retry do River
  }
  ```
- **Retry**: `river.WorkerDefaults` já fornece 3 tentativas com backoff exponencial

### Modificação em `internal/routines/mensalidadesEscoteiro.go`

- **O que muda**: A função original vira uma função de **enqueue** em vez de execução:
  ```go
  func EnqueueScoutMonthlyFees(ctx context.Context, client *river.Client[pgx.Tx]) error {
      params := make([]river.InsertManyParams, 0, len(contributors))
      for name, phone := range contributors {
          msg := buildMessage(name)
          params = append(params, river.InsertManyParams{
              Args: riverqueue.SendWhatsAppMessageArgs{
                  RecipientPhone: phone,
                  RecipientName:  name,
                  Message:        msg,
              },
          })
      }
      _, err := client.InsertMany(ctx, params)
      return err
  }
  ```

### `docker-compose.yml` — service PostgreSQL

```yaml
postgres:
  image: postgres:16-alpine
  container_name: kaizen-postgres
  environment:
    - POSTGRES_USER=kaizen
    - POSTGRES_PASSWORD=kaizen
    - POSTGRES_DB=kaizen_secretary
  ports:
    - "5432:5432"
  volumes:
    - postgres_data:/var/lib/postgresql/data
  networks:
    - manda-pra-mim
```

---

## Tech Decisions

| Decisão | Escolha | Rationale |
|---|---|---|
| River no mesmo processo | Sim (goroutine) | Simples para demo; em produção real seria processo separado |
| Migration automática | `rivermigrate` no startup | Evita passo manual; ideal para demo |
| Fan-out granularidade | 1 job por destinatário | Permite retry individual por mensagem |
| pgx driver | `pgx/v5` | Driver oficial recomendado pelo River |

---

## Dependencies a adicionar

```
github.com/riverqueue/river
github.com/riverqueue/river/riverdriver/riverpgxv5
github.com/jackc/pgx/v5
```

## Variáveis de ambiente a adicionar ao `.env.example`

```
DATABASE_URL=postgres://kaizen:kaizen@localhost:5432/kaizen_secretary
```
