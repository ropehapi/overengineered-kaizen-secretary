# Flagsmith — Especificação

## Problem Statement

Atualmente, para habilitar ou desabilitar uma rotina no worker é necessário alterar o código e fazer redeploy. Para fins educacionais, queremos demonstrar o padrão de feature flags com Flagsmith, separando o conceito de **deploy** (colocar código em produção) de **release** (ativar funcionalidade para usuários/ambientes).

## Goals

- [ ] Integrar o SDK do Flagsmith para consulta de feature flags em runtime
- [ ] Usar flags para controlar execução de rotinas (enable/disable sem redeploy)
- [ ] Usar um flag de valor para modo dry-run (não envia mensagens reais)
- [ ] Flagsmith self-hosted no docker-compose para demo reproduzível

## Out of Scope

| Feature | Razão |
|---|---|
| Segmentação de usuários / targeting | Sem usuários neste worker |
| Remote evaluation (server-side) | Local evaluation é suficiente para demo |
| A/B testing de conteúdo de mensagem | Adiciona complexidade além do escopo educacional |

---

## User Stories

### P1: Flagsmith SDK inicializado no worker ⭐ MVP

**User Story**: Como desenvolvedor, quero que o worker inicialize o cliente Flagsmith ao subir, para que todas as rotinas possam consultar flags em runtime.

**Acceptance Criteria**:

1. WHEN o worker inicia THEN o cliente Flagsmith SHALL ser inicializado com a `FLAGSMITH_API_KEY` do ambiente
2. WHEN `FLAGSMITH_HOST` está definido THEN o SDK SHALL apontar para esse host (self-hosted)
3. WHEN o Flagsmith não está acessível THEN o worker SHALL logar warning e continuar (fallback: todas as flags `false`)
4. WHEN `FLAGSMITH_API_KEY` não está definida THEN o worker SHALL logar erro e encerrar

**Independent Test**: Worker sobe com cliente Flagsmith inicializado e loga "flagsmith initialized" sem erros.

---

### P1: Flag de enable/disable por rotina ⭐ MVP

**User Story**: Como operador, quero controlar se a rotina `RememberScoutMonthlyFees` executa sem fazer redeploy, para demonstrar separação entre deploy e release.

**Acceptance Criteria**:

1. WHEN a rotina é disparada pelo cron THEN SHALL consultar flag `scout_monthly_reminder_enabled`
2. WHEN a flag está `true` THEN a rotina SHALL executar normalmente
3. WHEN a flag está `false` THEN a rotina SHALL ser pulada com log `"routine disabled by feature flag"`
4. WHEN a flag não existe no Flagsmith THEN SHALL usar fallback `false` (seguro por padrão)

**Independent Test**: Desabilitar a flag no Flagsmith UI, acionar a rotina, e verificar que o log "routine disabled" aparece sem enviar mensagens.

---

### P1: Flag de dry-run ⭐ MVP

**User Story**: Como operador, quero um modo dry-run ativável por flag, para que a rotina execute a lógica sem enviar mensagens reais (útil para validar em staging).

**Acceptance Criteria**:

1. WHEN a flag `dry_run_mode` está `true` THEN a rotina SHALL logar as mensagens que seriam enviadas, mas NÃO chamar o messaging-officer
2. WHEN a flag `dry_run_mode` está `false` THEN a rotina SHALL executar normalmente com envio real
3. WHEN em dry-run, o log SHALL incluir `"[DRY RUN]"` e o conteúdo da mensagem que seria enviada

**Independent Test**: Ativar `dry_run_mode` no Flagsmith UI, acionar rotina, verificar logs `[DRY RUN]` sem chamadas HTTP ao messaging-officer.

---

### P2: Flagsmith self-hosted no docker-compose

**User Story**: Como desenvolvedor, quero subir o Flagsmith com `docker compose up`, para que a demo seja reproduzível sem conta no SaaS.

**Acceptance Criteria**:

1. WHEN `docker compose up` é executado THEN Flagsmith UI SHALL estar disponível em `localhost:8000`
2. WHEN Flagsmith sobe THEN o worker SHALL se conectar automaticamente via `FLAGSMITH_HOST`

**Independent Test**: `curl http://localhost:8000` retorna 200. Worker loga "flagsmith initialized" apontando para o host local.

---

## Requirement Traceability

| ID | Story | Status |
|---|---|---|
| FLAG-01 | P1: SDK inicializado | Pending |
| FLAG-02 | P1: Flag enable/disable por rotina | Pending |
| FLAG-03 | P1: Flag dry-run | Pending |
| FLAG-04 | P2: Flagsmith no docker-compose | Pending |

## Success Criteria

- [ ] Worker sobe e inicializa cliente Flagsmith sem erros
- [ ] Desabilitar flag no UI impede execução da rotina (sem redeploy)
- [ ] Ativar dry-run no UI faz rotina logar sem enviar mensagens
- [ ] Stack completo sobe com `docker compose up`
