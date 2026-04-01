# Uber Zap — Especificação

## Problem Statement

O worker usa `slog` (stdlib do Go 1.21+) para logging estruturado. Para fins educacionais, queremos substituir pelo Uber Zap para demonstrar a evolução histórica de logging em Go (`fmt.Println` → `slog` → `zap`), os trade-offs de performance (zero-allocation), e a API fluente com campos tipados.

## Goals

- [ ] Substituir `slog` por `zap.Logger` em toda a aplicação
- [ ] Demonstrar logging estruturado com campos tipados (`zap.String`, `zap.Duration`, `zap.Error`)
- [ ] Configurar output JSON em produção e output legível (`zap.NewDevelopment`) em desenvolvimento

## Out of Scope

| Feature | Razão |
|---|---|
| Log rotation / arquivo de saída | Fora do escopo de demonstração |
| Integração com sistemas externos (Datadog, etc.) | Adiciona complexidade sem valor didático aqui |
| Sampling de logs | Desnecessário para o volume deste worker |

---

## User Stories

### P1: Logger Zap inicializado globalmente ⭐ MVP

**User Story**: Como desenvolvedor, quero que o worker inicialize um `zap.Logger` ao subir e o disponibilize globalmente, para que todas as partes da aplicação possam logar sem precisar passar o logger como parâmetro.

**Acceptance Criteria**:

1. WHEN o worker inicia THEN `zap.Logger` SHALL ser inicializado antes de qualquer outra operação
2. WHEN `APP_ENV=production` THEN SHALL usar `zap.NewProduction()` (JSON, sem cores)
3. WHEN `APP_ENV` não está definido ou é diferente de `production` THEN SHALL usar `zap.NewDevelopment()` (legível, com cores)
4. WHEN o worker encerra THEN `logger.Sync()` SHALL ser chamado para flush do buffer

**Independent Test**: Subir o worker e verificar que os logs aparecem em formato JSON (production) ou legível (development).

---

### P1: Substituir todos os logs do `internal/logger` por Zap ⭐ MVP

**User Story**: Como desenvolvedor, quero que o pacote `internal/logger` use Zap internamente, para que toda a aplicação migre sem alterações nos chamadores.

**Acceptance Criteria**:

1. WHEN `logger.Init()` é chamado THEN SHALL inicializar o Zap e configurar `zap.ReplaceGlobals`
2. WHEN qualquer parte do código loga usando o logger global THEN SHALL usar a instância Zap

**Independent Test**: `go build ./...` compila sem erros e logs aparecem no formato Zap.

---

### P1: Logs estruturados na rotina com campos tipados ⭐ MVP

**User Story**: Como desenvolvedor, quero que a rotina `RememberScoutMonthlyFees` use campos tipados do Zap, para demonstrar a diferença entre `fmt.Sprintf` e `zap.String/zap.Error`.

**Acceptance Criteria**:

1. WHEN uma mensagem é enviada com sucesso THEN SHALL logar com `zap.String("recipient", phone)` e `zap.String("status", "success")`
2. WHEN uma mensagem falha THEN SHALL logar com `zap.Error(err)` e `zap.String("recipient", phone)`
3. WHEN a rotina termina THEN SHALL logar summary com `zap.Int("sent", n)` e `zap.Int("failed", n)`

**Independent Test**: Verificar que os logs da rotina têm campos JSON estruturados (não strings interpoladas).

---

## Requirement Traceability

| ID | Story | Status |
|---|---|---|
| ZAP-01 | P1: Logger inicializado globalmente | Pending |
| ZAP-02 | P1: Substituir internal/logger | Pending |
| ZAP-03 | P1: Logs estruturados na rotina | Pending |

## Success Criteria

- [ ] `go build ./...` compila sem erros
- [ ] Logs em formato JSON quando `APP_ENV=production`
- [ ] Logs da rotina têm campos tipados (`recipient`, `status`, `error`)
- [ ] Nenhum `fmt.Println` ou `slog` remanescente na codebase
