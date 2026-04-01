# Uber Zap — Tasks

**Design**: `.specs/features/uber-zap/design.md`
**Status**: Draft

---

## Execution Plan

```
Phase 1 (Sequential — Foundation):
  T1 → T2

Phase 2 (Parallel — Substituição):
  T2 complete, então:
    ├── T3 [P]  (atualizar main.go)
    └── T4 [P]  (atualizar rotina)

Phase 3 (Sequential — Validação):
  T3 + T4 completos, então:
    T5
```

---

## Task Breakdown

### T1: Adicionar dependência `go.uber.org/zap` ao go.mod

**What**: Rodar `go get go.uber.org/zap` para adicionar o Zap ao projeto.
**Where**: `go.mod` / `go.sum`
**Depends on**: Nenhuma
**Requirement**: ZAP-01

**Done when**:
- [ ] `go.mod` contém `go.uber.org/zap`
- [ ] `go build ./...` compila sem erros

**Verify**:
```bash
go build ./...
```

**Commit**: `chore: add uber/zap dependency`

---

### T2: Reescrever `internal/logger/logger.go` com Zap

**What**: Substituir a implementação `slog` por `zap.Logger` mantendo a assinatura `Init()`.
**Where**: `internal/logger/logger.go` (reescrita)
**Depends on**: T1
**Requirement**: ZAP-01, ZAP-02

**Implementação**:
```go
package logger

import (
    "os"
    "go.uber.org/zap"
)

func Init() {
    var logger *zap.Logger
    var err error

    if os.Getenv("APP_ENV") == "production" {
        logger, err = zap.NewProduction()
    } else {
        logger, err = zap.NewDevelopment()
    }
    if err != nil {
        panic("failed to initialize zap logger: " + err.Error())
    }
    zap.ReplaceGlobals(logger)
}
```

**Done when**:
- [ ] `internal/logger/logger.go` usa `zap` em vez de `slog`
- [ ] `Init()` chama `zap.ReplaceGlobals`
- [ ] `APP_ENV=production` → `zap.NewProduction()` (JSON compacto)
- [ ] `APP_ENV` ausente → `zap.NewDevelopment()` (legível, colorido)
- [ ] `go build ./...` compila sem erros

**Verify**:
```bash
go build ./...
go vet ./internal/logger/...
APP_ENV=production go run cmd/worker/main.go 2>&1 | head -3   # deve ser JSON
go run cmd/worker/main.go 2>&1 | head -3                       # deve ser legível
```

**Commit**: `feat(logger): replace slog with uber/zap`

---

### T3: Adicionar `defer zap.L().Sync()` em `cmd/worker/main.go` [P]

**What**: Adicionar flush do buffer Zap no encerramento do worker.
**Where**: `cmd/worker/main.go` (modificação mínima)
**Depends on**: T2
**Requirement**: ZAP-01

**O que adicionar** (logo após `logger.Init()`):
```go
defer func() { _ = zap.L().Sync() }()
```

> Nota: o erro de `Sync()` é suprimido intencionalmente — no Linux, `Sync()` em `os.Stderr` retorna `ENOTTY` e isso é esperado.

**Done when**:
- [ ] `defer zap.L().Sync()` presente após `logger.Init()`
- [ ] `go build ./cmd/worker` compila sem erros

**Verify**:
```bash
go build ./cmd/worker
```

**Commit**: `feat(logger): add zap sync on worker shutdown`

---

### T4: Substituir logs `slog` por `zap.L()` com campos tipados na rotina [P]

**What**: Atualizar todos os `slog.X(...)` em `mensalidadesEscoteiro.go` para `zap.L().X(...)` com campos tipados.
**Where**: `internal/routines/mensalidadesEscoteiro.go` (modificação)
**Depends on**: T2
**Requirement**: ZAP-03

**Mapeamento de substituições**:
```go
// ANTES:
slog.Info("sending message", "recipient", phone)
slog.Error("failed to send", "recipient", phone, "error", err)
slog.Info("routine finished", "sent", sent, "failed", failed)

// DEPOIS:
zap.L().Info("sending message", zap.String("recipient", phone))
zap.L().Error("failed to send", zap.String("recipient", phone), zap.Error(err))
zap.L().Info("routine finished", zap.Int("sent", sent), zap.Int("failed", failed))
```

**Done when**:
- [ ] Nenhum `slog.` remanescente no arquivo
- [ ] Todos os logs usam `zap.L()` com campos tipados (`zap.String`, `zap.Error`, `zap.Int`, `zap.Duration`)
- [ ] `go build ./...` compila sem erros

**Verify**:
```bash
go build ./...
grep -r "slog\." internal/ && echo "SLOG ENCONTRADO" || echo "OK - sem slog"
```

**Commit**: `feat(logger): update scout routine to use structured zap fields`

---

### T5: Validação — comparar output development vs. production

**What**: Verificar visualmente a diferença entre os dois modos de logging.
**Where**: Terminal (execução manual)
**Depends on**: T3, T4
**Requirement**: ZAP-01, ZAP-02, ZAP-03

**Passos de validação**:
1. `go run cmd/worker/main.go` → logs em modo development (colorido, legível)
2. `APP_ENV=production go run cmd/worker/main.go` → logs em JSON compacto
3. Verificar que campos `recipient`, `status`, `error` aparecem como chaves JSON
4. Confirmar que não há `slog` ou `fmt.Println` remanescentes

**Done when**:
- [ ] Development mode: logs legíveis com nível colorido
- [ ] Production mode: logs JSON com campos tipados
- [ ] `grep -r "slog\." .` retorna vazio (exceto go.sum)
- [ ] `grep -r "fmt\.Println" internal/` retorna vazio

**Verify**:
```bash
grep -r "slog\." internal/ cmd/ || echo "OK"
grep -r "fmt\.Println" internal/ cmd/ || echo "OK"
APP_ENV=production go run cmd/worker/main.go 2>&1 | head -5 | python3 -m json.tool
```

**Commit**: `test(logger): validate zap output in development and production modes`

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
    T5
```

---

## Task Granularity Check

| Task | Escopo | Status |
|---|---|---|
| T1: Adicionar dependência | `go.mod` | ✅ Granular |
| T2: Reescrever logger.go | 1 arquivo, reescrita | ✅ Granular |
| T3: Sync no main.go | 1 linha no main | ✅ Granular |
| T4: Substituir logs na rotina | 1 arquivo, substituição | ✅ Granular |
| T5: Validação comparativa | Teste manual | ✅ Granular |
