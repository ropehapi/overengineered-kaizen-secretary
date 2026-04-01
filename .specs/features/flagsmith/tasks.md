# Flagsmith — Tasks

**Design**: `.specs/features/flagsmith/design.md`
**Status**: Draft

---

## Execution Plan

```
Phase 1 (Sequential — Foundation):
  T1 → T2

Phase 2 (Parallel — Integração):
  T2 complete, então:
    ├── T3 [P]  (verificação de flag no main)
    ├── T4 [P]  (dry-run na rotina)
    └── T5 [P]  (Flagsmith no docker-compose)

Phase 3 (Sequential — Validação):
  T3 + T4 + T5 completos, então:
    T6 → T7
```

---

## Task Breakdown

### T1: Adicionar dependência do SDK Flagsmith ao go.mod

**What**: Rodar `go get` para adicionar o client Flagsmith Go ao projeto.
**Where**: `go.mod` / `go.sum`
**Depends on**: Nenhuma
**Requirement**: FLAG-01

**Pacote a adicionar**:
```
github.com/Flagsmith/flagsmith-go-client/v3
```

**Done when**:
- [ ] `go.mod` contém `github.com/Flagsmith/flagsmith-go-client/v3`
- [ ] `go build ./...` compila sem erros

**Verify**:
```bash
go build ./...
```

**Commit**: `chore: add flagsmith-go-client dependency`

---

### T2: Criar `internal/featureflags/flagsmith.go`

**What**: Implementar wrapper do SDK Flagsmith com `Init()`, `IsEnabled()` e `GetValue()`, com fallback fail-safe.
**Where**: `internal/featureflags/flagsmith.go` (novo arquivo)
**Depends on**: T1
**Reuses**: Padrão de `Init()` de `internal/logger/logger.go`
**Requirement**: FLAG-01

**Implementação esperada**:
```go
package featureflags

import (
    "os"
    flagsmith "github.com/Flagsmith/flagsmith-go-client/v3"
)

var (
    client    *flagsmith.Client
    available bool
)

func Init() error {
    apiKey := os.Getenv("FLAGSMITH_API_KEY")
    if apiKey == "" {
        return fmt.Errorf("FLAGSMITH_API_KEY not set")
    }
    opts := []flagsmith.Option{}
    if host := os.Getenv("FLAGSMITH_HOST"); host != "" {
        opts = append(opts, flagsmith.WithBaseURL(host))
    }
    client = flagsmith.NewClient(apiKey, opts...)
    available = true
    return nil
}

func IsEnabled(flagName string) bool {
    if !available { return false }
    flags, err := client.GetEnvironmentFlags()
    if err != nil { return false }
    enabled, _ := flags.IsFeatureEnabled(flagName)
    return enabled
}
```

**Done when**:
- [ ] Arquivo `internal/featureflags/flagsmith.go` criado
- [ ] `Init()` lê `FLAGSMITH_API_KEY` (erro se ausente) e `FLAGSMITH_HOST` (opcional)
- [ ] `IsEnabled()` retorna `false` se `available == false` (fail-safe)
- [ ] `go build ./...` compila sem erros

**Verify**:
```bash
go build ./...
go vet ./internal/featureflags/...
```

**Commit**: `feat(flags): create flagsmith wrapper package with fail-safe fallback`

---

### T3: Integrar verificação de flag em `cmd/worker/main.go` [P]

**What**: Inicializar Flagsmith no startup e envolver chamada de rotina com verificação de flag `scout_monthly_reminder_enabled`.
**Where**: `cmd/worker/main.go` (modificação)
**Depends on**: T2
**Requirement**: FLAG-01, FLAG-02

**O que adicionar** (após `logger.Init()`):
```go
if err := featureflags.Init(); err != nil {
    zap.L().Warn("flagsmith unavailable, all flags default to false",
        zap.Error(err))
}

// No AddFunc:
c.AddFunc(expr, func() {
    if !featureflags.IsEnabled("scout_monthly_reminder_enabled") {
        zap.L().Info("routine disabled by feature flag",
            zap.String("routine", "RememberScoutMonthlyFees"))
        return
    }
    routines.RememberScoutMonthlyFees()
})
```

**Done when**:
- [ ] `featureflags.Init()` chamado no startup com warn em falha (não panic)
- [ ] Flag `scout_monthly_reminder_enabled` verificada antes de chamar a rotina
- [ ] Log `"routine disabled by feature flag"` quando flag está `false`
- [ ] `go build ./cmd/worker` compila sem erros

**Verify**:
```bash
go build ./cmd/worker
```

**Commit**: `feat(flags): gate scout routine behind scout_monthly_reminder_enabled flag`

---

### T4: Adicionar dry-run mode em `mensalidadesEscoteiro.go` [P]

**What**: Verificar flag `dry_run_mode` por envio e logar em vez de chamar o messaging-officer.
**Where**: `internal/routines/mensalidadesEscoteiro.go` (modificação)
**Depends on**: T2
**Requirement**: FLAG-03

**O que adicionar por iteração**:
```go
if featureflags.IsEnabled("dry_run_mode") {
    zap.L().Info("[DRY RUN] would send message",
        zap.String("recipient_phone", phone),
        zap.String("recipient_name", name),
        zap.String("message_preview", msgBody[:min(50, len(msgBody))]))
    continue
}
// ... lógica de envio real existente ...
```

**Done when**:
- [ ] Verificação `dry_run_mode` presente antes de cada chamada HTTP
- [ ] Log `[DRY RUN]` com `recipient_phone`, `recipient_name` e preview da mensagem
- [ ] Quando dry-run ativo, nenhuma chamada HTTP ao messaging-officer ocorre
- [ ] `go build ./...` compila sem erros

**Verify**:
```bash
go build ./...
go vet ./internal/routines/...
```

**Commit**: `feat(flags): add dry_run_mode support to scout monthly routine`

---

### T5: Adicionar Flagsmith ao `docker-compose.yml` [P]

**What**: Adicionar service `flagsmith` com imagem oficial e porta 8000 exposta.
**Where**: `docker-compose.yml` (modificação)
**Depends on**: T2
**Requirement**: FLAG-04

**Service a adicionar**:
```yaml
flagsmith:
  image: flagsmith/flagsmith:latest
  container_name: kaizen-flagsmith
  ports:
    - "8000:8000"
  environment:
    - DJANGO_ALLOWED_HOSTS=*
  networks:
    - manda-pra-mim
```

**Variáveis a adicionar no service `app`**:
```yaml
environment:
  - FLAGSMITH_API_KEY=${FLAGSMITH_API_KEY}
  - FLAGSMITH_HOST=http://flagsmith:8000/api/v1
```

**Atualizar `.env.example`**:
```
FLAGSMITH_API_KEY=sua-env-key-aqui
```

**Done when**:
- [ ] Service `flagsmith` adicionado ao compose
- [ ] Service `app` tem as variáveis `FLAGSMITH_API_KEY` e `FLAGSMITH_HOST`
- [ ] `.env.example` atualizado com `FLAGSMITH_API_KEY`
- [ ] `docker compose config` valida sem erros

**Verify**:
```bash
docker compose config
docker compose up flagsmith -d
sleep 10
curl -s http://localhost:8000/api/v1/flags/ | grep -i "results\|detail" && echo "Flagsmith OK"
docker compose down
```

**Commit**: `feat(flags): add Flagsmith self-hosted to docker-compose`

---

### T6: Setup inicial do Flagsmith — criar projeto e flags

**What**: Acessar o Flagsmith UI, criar o projeto `kaizen-secretary` e as duas flags necessárias.
**Where**: `http://localhost:8000` (setup manual pós-deploy)
**Depends on**: T5
**Requirement**: FLAG-02, FLAG-03

**Passos**:
1. `docker compose up flagsmith -d`
2. Acessar `http://localhost:8000`
3. Criar conta admin (primeira vez)
4. Criar organização `Kaizen` e projeto `kaizen-secretary`
5. Criar Environment `Development` → copiar API Key para o `.env`
6. Criar flag `scout_monthly_reminder_enabled` (boolean, default `true`)
7. Criar flag `dry_run_mode` (boolean, default `false`)

**Done when**:
- [ ] Projeto `kaizen-secretary` criado no Flagsmith UI
- [ ] API Key copiada para `.env` como `FLAGSMITH_API_KEY`
- [ ] Flag `scout_monthly_reminder_enabled` criada e habilitada
- [ ] Flag `dry_run_mode` criada e desabilitada
- [ ] Worker sobe e loga "flagsmith initialized" (sem erro)

**Verify**:
```bash
docker compose up --build
# Verificar nos logs do worker: sem erros de Flagsmith
```

**Commit**: `docs: document flagsmith setup steps in README or docs/`

---

### T7: Validação E2E — enable/disable e dry-run via UI

**What**: Demonstrar o ciclo completo de feature flags sem redeploy.
**Where**: Flagsmith UI + logs do worker
**Depends on**: T3, T4, T6
**Requirement**: FLAG-02, FLAG-03

**Passos de validação**:

1. **Teste 1 — Disable flag**:
   - No Flagsmith UI, desabilitar `scout_monthly_reminder_enabled`
   - Alterar cron para rodar em 10s temporariamente
   - Verificar log: `"routine disabled by feature flag"`
   - Reabilitar a flag

2. **Teste 2 — Dry-run**:
   - Habilitar `dry_run_mode` no Flagsmith UI
   - Acionar rotina
   - Verificar logs: `[DRY RUN] would send message` para cada destinatário
   - Verificar que nenhum HTTP ao messaging-officer foi feito
   - Desabilitar `dry_run_mode`

3. Reverter cron para expressão original

**Done when**:
- [ ] Teste 1: Log `"routine disabled"` aparece quando flag está off
- [ ] Teste 2: Logs `[DRY RUN]` aparecem para cada destinatário
- [ ] Nenhuma mensagem real enviada durante dry-run
- [ ] Cron revertido para expressão original

**Commit**: `test(flags): validate feature flag enable/disable and dry-run via Flagsmith UI`

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
| T2: Criar featureflags/flagsmith.go | 1 arquivo novo | ✅ Granular |
| T3: Flag gate no main.go | 1 modificação focada | ✅ Granular |
| T4: Dry-run na rotina | 1 arquivo, 1 verificação | ✅ Granular |
| T5: Flagsmith no compose | 1 arquivo, 1 service | ✅ Granular |
| T6: Setup manual UI | Configuração única | ✅ Granular |
| T7: Validação E2E | Teste manual | ✅ Granular |
