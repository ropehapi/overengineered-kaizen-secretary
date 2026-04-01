# RiverQueue — Tasks

**Design**: `.specs/features/river-queue/design.md`
**Status**: Draft

---

## Execution Plan

```
Phase 1 (Sequential — Foundation):
  T1 → T2 → T3

Phase 2 (Parallel — Core):
  T3 complete, então:
    ├── T4 [P]  (definir args do job)
    └── T5 [P]  (PostgreSQL no docker-compose)

Phase 3 (Sequential — Worker e Enqueue):
  T4 complete, então:
    T6 → T7 → T8

Phase 4 (Sequential — Integração e Validação):
  T7 + T8 + T5 completos, então:
    T9 → T10
```

---

## Task Breakdown

### T1: Adicionar dependências River + pgx ao go.mod

**What**: Rodar `go get` para adicionar RiverQueue e o driver pgx ao projeto.
**Where**: `go.mod` / `go.sum`
**Depends on**: Nenhuma
**Requirement**: RIVER-01

**Pacotes a adicionar**:
```
github.com/riverqueue/river
github.com/riverqueue/river/riverdriver/riverpgxv5
github.com/jackc/pgx/v5
```

**Done when**:
- [ ] `go.mod` contém os 3 pacotes
- [ ] `go build ./...` compila sem erros

**Verify**:
```bash
go build ./...
```

**Commit**: `chore: add riverqueue and pgx dependencies`

---

### T2: Criar `internal/riverqueue/jobs.go` — definição do tipo de job

**What**: Definir `SendWhatsAppMessageArgs` com campos e método `Kind()` exigido pelo River.
**Where**: `internal/riverqueue/jobs.go` (novo arquivo)
**Depends on**: T1
**Requirement**: RIVER-02

**Implementação**:
```go
package riverqueue

type SendWhatsAppMessageArgs struct {
    RecipientPhone string `json:"recipient_phone"`
    RecipientName  string `json:"recipient_name"`
    Message        string `json:"message"`
}

func (SendWhatsAppMessageArgs) Kind() string {
    return "send_whatsapp_message"
}
```

**Done when**:
- [ ] Arquivo `internal/riverqueue/jobs.go` criado
- [ ] `SendWhatsAppMessageArgs` tem os 3 campos JSON
- [ ] Método `Kind()` retorna `"send_whatsapp_message"`
- [ ] `go build ./...` compila sem erros

**Verify**:
```bash
go build ./...
go vet ./internal/riverqueue/...
```

**Commit**: `feat(river): define SendWhatsAppMessageArgs job type`

---

### T3: Criar `internal/riverqueue/worker.go` — worker que processa o job

**What**: Implementar `SendWhatsAppMessageWorker` que chama o messaging-officer para um destinatário.
**Where**: `internal/riverqueue/worker.go` (novo arquivo)
**Depends on**: T2
**Reuses**: Lógica de HTTP POST de `internal/routines/mensalidadesEscoteiro.go`
**Requirement**: RIVER-02, RIVER-04

**Implementação**:
```go
package riverqueue

import (
    "context"
    "fmt"
    "net/http"
    "os"

    "github.com/riverqueue/river"
)

type SendWhatsAppMessageWorker struct {
    river.WorkerDefaults[SendWhatsAppMessageArgs]
}

func (w *SendWhatsAppMessageWorker) Work(ctx context.Context, job *river.Job[SendWhatsAppMessageArgs]) error {
    args := job.Args
    host := os.Getenv("MESSAGING_OFFICER_HOST")
    port := os.Getenv("MESSAGING_OFFICER_PORT")
    // ... monta payload JSON e faz POST para messaging-officer ...
    // Retorna error em falha → River faz retry automático
    return nil
}
```

**Done when**:
- [ ] `SendWhatsAppMessageWorker` implementa interface `river.Worker`
- [ ] `Work()` lê `MESSAGING_OFFICER_HOST` e `MESSAGING_OFFICER_PORT` do ambiente
- [ ] `Work()` retorna `error` em falha HTTP (aciona retry do River)
- [ ] `go build ./...` compila sem erros

**Verify**:
```bash
go build ./...
go vet ./internal/riverqueue/...
```

**Commit**: `feat(river): implement SendWhatsAppMessageWorker`

---

### T4: Criar `internal/riverqueue/setup.go` — inicialização do River [P]

**What**: Criar função `Init()` que conecta ao PostgreSQL, roda migration do River e retorna client + workers.
**Where**: `internal/riverqueue/setup.go` (novo arquivo)
**Depends on**: T3
**Requirement**: RIVER-01, RIVER-04

**Implementação esperada**:
```go
package riverqueue

import (
    "context"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/riverqueue/river"
    "github.com/riverqueue/river/riverdriver/riverpgxv5"
    "github.com/riverqueue/river/rivermigrate"
)

func Init(ctx context.Context, pool *pgxpool.Pool) (*river.Client[pgx.Tx], *river.Workers, error) {
    // 1. Rodar migrations do River schema
    migrator, _ := rivermigrate.New(riverpgxv5.New(pool), nil)
    migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)

    // 2. Registrar workers
    workers := river.NewWorkers()
    river.AddWorker(workers, &SendWhatsAppMessageWorker{})

    // 3. Criar client
    client, _ := river.NewClient(riverpgxv5.New(pool), &river.Config{
        Queues: map[string]river.QueueConfig{
            river.QueueDefault: {MaxWorkers: 10},
        },
        Workers: workers,
    })
    return client, workers, nil
}
```

**Done when**:
- [ ] `Init()` aceita `ctx` e `*pgxpool.Pool`
- [ ] Migration do River schema roda no startup
- [ ] `SendWhatsAppMessageWorker` registrado nos workers
- [ ] Retorna `*river.Client` e `*river.Workers`
- [ ] `go build ./...` compila sem erros

**Verify**:
```bash
go build ./...
go vet ./internal/riverqueue/...
```

**Commit**: `feat(river): create river initialization with auto-migration`

---

### T5: Adicionar PostgreSQL ao `docker-compose.yml` [P]

**What**: Adicionar service `postgres` com volume persistente e variável `DATABASE_URL` no worker.
**Where**: `docker-compose.yml` (modificação)
**Depends on**: T1
**Requirement**: RIVER-01

**Service a adicionar**:
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

volumes:
  postgres_data:
```

**Variável a adicionar no service `app`**:
```yaml
environment:
  - DATABASE_URL=postgres://kaizen:kaizen@postgres:5432/kaizen_secretary
```

**Atualizar `.env.example`**:
```
DATABASE_URL=postgres://kaizen:kaizen@localhost:5432/kaizen_secretary
```

**Done when**:
- [ ] Service `postgres` adicionado com usuário, senha e database configurados
- [ ] Volume `postgres_data` declarado
- [ ] `DATABASE_URL` no service `app` apontando para `postgres` (hostname interno)
- [ ] `.env.example` atualizado
- [ ] `docker compose config` valida sem erros

**Verify**:
```bash
docker compose config
docker compose up postgres -d
sleep 3
docker compose exec postgres psql -U kaizen -d kaizen_secretary -c "\l"
docker compose down
```

**Commit**: `feat(river): add PostgreSQL to docker-compose for River backend`

---

### T6: Refatorar `mensalidadesEscoteiro.go` — extrair lógica de enqueue

**What**: Transformar `RememberScoutMonthlyFees()` em `EnqueueScoutMonthlyFees()` que insere jobs em lote no River.
**Where**: `internal/routines/mensalidadesEscoteiro.go` (modificação)
**Depends on**: T4
**Reuses**: Map `contributors` e função `buildMessage` existentes
**Requirement**: RIVER-03

**O que muda**:
```go
// ANTES: função executa envio diretamente
func RememberScoutMonthlyFees() { /* envia HTTP */ }

// DEPOIS: função insere jobs no River
func EnqueueScoutMonthlyFees(ctx context.Context, client *river.Client[pgx.Tx]) error {
    params := make([]river.InsertManyParams, 0, len(contributors))
    for name, phone := range contributors {
        params = append(params, river.InsertManyParams{
            Args: riverqueue.SendWhatsAppMessageArgs{
                RecipientPhone: phone,
                RecipientName:  name,
                Message:        buildMessage(name),
            },
        })
    }
    _, err := client.InsertMany(ctx, params)
    if err != nil {
        return fmt.Errorf("failed to enqueue jobs: %w", err)
    }
    zap.L().Info("jobs enqueued", zap.Int("count", len(params)),
        zap.String("routine", "RememberScoutMonthlyFees"))
    return err
}
```

**Done when**:
- [ ] `EnqueueScoutMonthlyFees(ctx, client)` substitui `RememberScoutMonthlyFees()`
- [ ] Jobs inseridos em lote com `InsertMany` (atômico)
- [ ] Log `"N jobs enqueued"` após inserção bem-sucedida
- [ ] Nenhuma chamada HTTP ao messaging-officer neste arquivo
- [ ] `go build ./...` compila sem erros

**Verify**:
```bash
go build ./...
grep -n "http\." internal/routines/mensalidadesEscoteiro.go || echo "OK - sem HTTP na rotina"
```

**Commit**: `feat(river): refactor scout routine to enqueue River jobs instead of direct HTTP`

---

### T7: Integrar River em `cmd/worker/main.go`

**What**: Conectar ao PostgreSQL, inicializar River (client + workers), e atualizar `AddFunc` para usar `EnqueueScoutMonthlyFees`.
**Where**: `cmd/worker/main.go` (modificação)
**Depends on**: T4, T6
**Requirement**: RIVER-01, RIVER-03, RIVER-04

**O que adicionar**:
```go
// Conectar ao PostgreSQL
pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
if err != nil {
    zap.L().Fatal("failed to connect to database", zap.Error(err))
}
defer pool.Close()

// Inicializar River
riverClient, riverWorkers, err := riverqueue.Init(ctx, pool)
if err != nil {
    zap.L().Fatal("failed to initialize river", zap.Error(err))
}

// Iniciar workers
if err := riverWorkers.Start(ctx); err != nil {
    zap.L().Fatal("failed to start river workers", zap.Error(err))
}
defer riverWorkers.Stop(ctx)

// Registrar cron usando enqueue
c.AddFunc(expr, func() {
    if err := routines.EnqueueScoutMonthlyFees(ctx, riverClient); err != nil {
        zap.L().Error("failed to enqueue routine", zap.Error(err))
    }
})
```

**Done when**:
- [ ] Pool pgx criado com `DATABASE_URL`
- [ ] `riverqueue.Init()` chamado, retornando client e workers
- [ ] `riverWorkers.Start(ctx)` chamado antes do `c.Start()`
- [ ] `defer riverWorkers.Stop(ctx)` registrado
- [ ] `AddFunc` usa `routines.EnqueueScoutMonthlyFees`
- [ ] `go build ./cmd/worker` compila sem erros

**Verify**:
```bash
go build ./cmd/worker
```

**Commit**: `feat(river): integrate River client and workers into main worker process`

---

### T8: Validação — jobs criados e processados no banco

**What**: Subir o stack completo e verificar que os jobs são inseridos e processados no PostgreSQL.
**Where**: `docker-compose.yml` stack + psql
**Depends on**: T5, T7
**Requirement**: RIVER-01, RIVER-02, RIVER-03, RIVER-04

**Passos de validação**:
1. `docker compose up --build`
2. Alterar cron para `"*/30 * * * * *"` temporariamente
3. Aguardar disparo
4. Verificar jobs no banco:
```sql
SELECT kind, state, attempt, created_at
FROM river_jobs
ORDER BY created_at DESC
LIMIT 20;
```
5. Verificar que `state = 'completed'` para jobs processados
6. Forçar falha (parar messaging-officer) e verificar `state = 'retryable'`
7. Reverter cron para expressão original

**Done when**:
- [ ] `river_jobs` contém registros após disparo do cron
- [ ] Jobs processados aparecem com `state = 'completed'`
- [ ] Sem messaging-officer: jobs aparecem com `state = 'retryable'`
- [ ] Cron revertido para expressão original

**Verify**:
```bash
docker compose exec postgres psql -U kaizen -d kaizen_secretary \
  -c "SELECT kind, state, count(*) FROM river_jobs GROUP BY kind, state;"
```

**Commit**: `test(river): validate job enqueueing and processing in PostgreSQL`

---

## Parallel Execution Map

```
Phase 1 (Sequential):
  T1 ──→ T2 ──→ T3

Phase 2 (Parallel):
  T3 complete, então:
    ├── T4 [P]
    └── T5 [P]

Phase 3 (Sequential):
  T4 complete, então:
    T6 ──→ T7

Phase 4 (Sequential):
  T7 + T5 completos, então:
    T8
```

---

## Task Granularity Check

| Task | Escopo | Status |
|---|---|---|
| T1: Dependências | `go.mod` | ✅ Granular |
| T2: jobs.go | 1 arquivo, tipo de dados | ✅ Granular |
| T3: worker.go | 1 arquivo, 1 worker | ✅ Granular |
| T4: setup.go | 1 arquivo, inicialização | ✅ Granular |
| T5: PostgreSQL no compose | 1 service | ✅ Granular |
| T6: Refatorar rotina | 1 arquivo, lógica de enqueue | ✅ Granular |
| T7: Integrar no main | 1 arquivo, wiring | ✅ Granular |
| T8: Validação E2E | Teste com psql | ✅ Granular |
