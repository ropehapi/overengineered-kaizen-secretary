# Uber Zap — Design

**Spec**: `.specs/features/uber-zap/spec.md`
**Status**: Draft

---

## Architecture Overview

Substituição direta de `slog` por `zap.Logger` no pacote `internal/logger`. O restante da codebase não muda de interface — apenas os chamadores dentro das rotinas são atualizados para usar campos tipados.

```
internal/logger/logger.go
  ├── ANTES: slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, ...)))
  └── DEPOIS: zap.ReplaceGlobals(logger) + zap.L() disponível globalmente

internal/routines/mensalidadesEscoteiro.go
  ├── ANTES: slog.Info("mensagem enviada", "recipient", phone)
  └── DEPOIS: zap.L().Info("mensagem enviada", zap.String("recipient", phone))
```

**Decisão de logger global vs. injetado:**
Para fins educacionais, usar `zap.L()` (global) é mais simples e didático. O padrão de injeção de dependência é deixado como ponto de discussão na apresentação.

---

## Code Reuse Analysis

### Existing Components to Leverage

| Component | Location | How to Use |
|---|---|---|
| `internal/logger/logger.go` | `internal/logger/logger.go` | Reescrever `Init()` mantendo a mesma assinatura |
| Chamadas de log na rotina | `internal/routines/mensalidadesEscoteiro.go` | Substituir `slog.X` por `zap.L().X` com campos tipados |

### Integration Points

| Sistema | Método |
|---|---|
| `cmd/worker/main.go` | Adicionar `defer zap.L().Sync()` após `logger.Init()` |
| Rotina existente | Substituição de chamadas de log (find/replace guiado) |

---

## Components

### `internal/logger/logger.go` (reescrita)

- **Purpose**: Inicializar `zap.Logger` baseado no ambiente e registrá-lo como logger global
- **Location**: `internal/logger/logger.go`
- **Interface** (mantida):
  - `Init()` — sem retorno, configura o logger global
- **Implementação**:
  ```go
  func Init() {
      var logger *zap.Logger
      if os.Getenv("APP_ENV") == "production" {
          logger, _ = zap.NewProduction()
      } else {
          logger, _ = zap.NewDevelopment()
      }
      zap.ReplaceGlobals(logger)
  }
  ```

### Modificação em `cmd/worker/main.go`

- **O que muda**: Adicionar `defer zap.L().Sync()` imediatamente após `logger.Init()`
- **Por quê**: Zap usa buffer interno; `Sync()` garante que todos os logs são escritos antes do processo encerrar

### Modificação em `internal/routines/mensalidadesEscoteiro.go`

- **Purpose**: Substituir chamadas `slog.X(...)` por `zap.L().X(...)` com campos tipados
- **Campos tipados a usar**:
  - `zap.String("recipient", phone)`
  - `zap.String("recipient_name", name)`
  - `zap.Error(err)`
  - `zap.Int("sent", sentCount)`
  - `zap.Int("failed", failedCount)`
  - `zap.Duration("elapsed", elapsed)`

---

## Tech Decisions

| Decisão | Escolha | Rationale |
|---|---|---|
| Logger global vs. injetado | Global (`zap.L()`) | Mais simples para demo; injeção é ponto de discussão |
| Production vs. development config | Via `APP_ENV` env var | Padrão comum em Go, fácil de demonstrar |
| `zap.NewProduction` vs. custom | `NewProduction` / `NewDevelopment` | Defaults sensatos, sem boilerplate desnecessário |

---

## Dependencies a adicionar

```
go.uber.org/zap
```
