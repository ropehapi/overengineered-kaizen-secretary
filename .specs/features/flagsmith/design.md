# Flagsmith — Design

**Spec**: `.specs/features/flagsmith/spec.md`
**Status**: Draft

---

## Architecture Overview

```
cmd/worker/main.go
  └── flagsmith.Init() ──────────────────────────────────────────┐
                                                                  │
  c.AddFunc(expr, func() {                                        │
      if !flagsmith.IsEnabled("scout_monthly_reminder_enabled") { │
          zap.L().Info("routine disabled by feature flag")        │
          return                                                   │
      }                                                           │
      routines.RememberScoutMonthlyFees()                        │
  })                                                              │
                                                                  ▼
                                               ┌──────────────────────────────┐
                                               │  internal/featureflags/      │
                                               │  flagsmith.go                │
                                               │                              │
                                               │  - Inicializa SDK client     │
                                               │  - IsEnabled(flagName) bool  │
                                               │  - GetValue(flagName) string │
                                               └──────────────────────────────┘
                                                          │ HTTP
                                                          ▼
                                               ┌──────────────────────────────┐
                                               │   Flagsmith (docker-compose) │
                                               │   porta 8000 (UI + API)      │
                                               └──────────────────────────────┘

Fluxo dentro da rotina (dry-run):
  RememberScoutMonthlyFees()
    └── if featureflags.IsEnabled("dry_run_mode")
          → loga "[DRY RUN] would send to <phone>"
          → NÃO chama messaging-officer
        else
          → chama messaging-officer normalmente
```

---

## Code Reuse Analysis

### Existing Components to Leverage

| Component | Location | How to Use |
|---|---|---|
| Padrão `Init()` | `internal/logger/logger.go` | Replicar para `internal/featureflags/flagsmith.go` |
| Rotina existente | `internal/routines/mensalidadesEscoteiro.go` | Adicionar verificações de flag antes de enviar |
| `os.Getenv` pattern | `cmd/worker/main.go` | Ler `FLAGSMITH_API_KEY` e `FLAGSMITH_HOST` |

### Integration Points

| Sistema | Método |
|---|---|
| `cmd/worker/main.go` | Chamar `featureflags.Init()` no startup + verificar flag antes de chamar rotina |
| `internal/routines/` | Consultar `featureflags.IsEnabled("dry_run_mode")` por iteração |
| `docker-compose.yml` | Adicionar service `flagsmith` |

---

## Components

### `internal/featureflags/flagsmith.go`

- **Purpose**: Encapsular o SDK do Flagsmith com fallback seguro em caso de falha de conexão
- **Location**: `internal/featureflags/flagsmith.go` (novo arquivo)
- **Interfaces**:
  - `Init() error` — inicializa o client com `FLAGSMITH_API_KEY` e `FLAGSMITH_HOST`
  - `IsEnabled(flagName string) bool` — retorna `false` se Flagsmith inacessível (fail-safe)
  - `GetValue(flagName string) string` — retorna string value de um flag (ex: template de mensagem)
- **Implementação de fallback**:
  ```go
  var client *flagsmith.Client
  var available bool  // false se Init falhou

  func IsEnabled(flagName string) bool {
      if !available {
          return false  // fail-safe: flags desabilitadas se Flagsmith down
      }
      flags, err := client.GetEnvironmentFlags()
      if err != nil {
          return false
      }
      enabled, _ := flags.IsFeatureEnabled(flagName)
      return enabled
  }
  ```
- **Dependencies**: `github.com/Flagsmith/flagsmith-go-client/v3`

### Modificação em `cmd/worker/main.go`

- **O que muda**:
  ```go
  if err := featureflags.Init(); err != nil {
      zap.L().Warn("flagsmith unavailable, all flags default to false", zap.Error(err))
  }

  c.AddFunc(expr, func() {
      if !featureflags.IsEnabled("scout_monthly_reminder_enabled") {
          zap.L().Info("routine disabled by feature flag",
              zap.String("routine", "RememberScoutMonthlyFees"))
          return
      }
      routines.RememberScoutMonthlyFees()
  })
  ```

### Modificação em `internal/routines/mensalidadesEscoteiro.go`

- **O que muda**: Adicionar verificação `dry_run_mode` por envio:
  ```go
  if featureflags.IsEnabled("dry_run_mode") {
      zap.L().Info("[DRY RUN] would send message",
          zap.String("recipient", phone),
          zap.String("message", msgBody))
      continue
  }
  // ... envio real ...
  ```

### `docker-compose.yml` — service Flagsmith

- **Imagem**: `flagsmith/flagsmith:latest`
- **Porta**: `8000` (UI + API REST)
- **Banco**: SQLite embutido (suficiente para demo)

---

## Tech Decisions

| Decisão | Escolha | Rationale |
|---|---|---|
| Self-hosted vs. SaaS | Self-hosted (docker-compose) | Demo reproduzível sem conta externa |
| Fail-safe vs. fail-open | Fail-safe (`false` quando down) | Mais seguro: rotina não roda se Flagsmith inacessível |
| Local vs. remote evaluation | Remote (API call por execução) | Mais simples; demonstra a chamada de rede |
| Banco do Flagsmith | SQLite embutido | Evita PostgreSQL adicional se River não for implementado junto |

---

## Flags a criar manualmente no Flagsmith UI após subir

| Flag Name | Type | Default | Descrição |
|---|---|---|---|
| `scout_monthly_reminder_enabled` | Boolean | `true` | Habilita/desabilita a rotina de lembretes |
| `dry_run_mode` | Boolean | `false` | Loga mensagens sem enviar |

---

## Dependencies a adicionar

```
github.com/Flagsmith/flagsmith-go-client/v3
```

## Variáveis de ambiente a adicionar ao `.env.example`

```
FLAGSMITH_API_KEY=sua-env-key-aqui
FLAGSMITH_HOST=http://localhost:8000/api/v1
```
