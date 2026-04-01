# RiverQueue — Especificação

## Problem Statement

Atualmente, o cron chama a rotina diretamente e de forma síncrona: se a aplicação reiniciar no meio de um envio, as mensagens restantes são perdidas. Para fins educacionais, queremos demonstrar o padrão de job queue com RiverQueue (PostgreSQL-backed), separando o **agendamento** (scheduler) da **execução** (workers), e adicionando durabilidade, retry automático e histórico de jobs.

## Goals

- [ ] Substituir execução direta de rotinas por inserção de jobs no RiverQueue
- [ ] Workers independentes processam os jobs com retry automático em falha
- [ ] PostgreSQL como backend de persistência dos jobs
- [ ] Cada envio de mensagem individual vira um job separado (fan-out)

## Out of Scope

| Feature | Razão |
|---|---|
| River Dashboard UI | Requer setup adicional sem valor didático proporcional |
| Priorização de filas | Desnecessário para o caso de uso |
| Jobs periódicos via River (substituir robfig/cron) | Manter o cron para mostrar integração, não substituição |

---

## User Stories

### P1: PostgreSQL no docker-compose ⭐ MVP

**User Story**: Como desenvolvedor, quero PostgreSQL disponível via docker-compose, para que o RiverQueue tenha um backend de persistência para a demo.

**Acceptance Criteria**:

1. WHEN `docker compose up` é executado THEN PostgreSQL SHALL estar disponível na porta `5432`
2. WHEN o worker sobe THEN SHALL se conectar ao PostgreSQL usando `DATABASE_URL` do ambiente
3. WHEN o banco sobe pela primeira vez THEN o schema do River SHALL ser criado automaticamente via migration

**Independent Test**: `psql $DATABASE_URL -c "\dt river_*"` lista as tabelas do River.

---

### P1: Job `SendWhatsAppMessageJob` definido ⭐ MVP

**User Story**: Como desenvolvedor, quero um tipo de job que encapsula o envio de uma mensagem WhatsApp, para que cada envio individual seja rastreável e retriável.

**Acceptance Criteria**:

1. WHEN o job é criado THEN SHALL conter os campos `RecipientPhone`, `RecipientName` e `Message`
2. WHEN o job é executado pelo worker THEN SHALL chamar o messaging-officer e retornar erro em falha
3. WHEN o job falha THEN River SHALL fazer retry automático com backoff exponencial (até 3 tentativas)
4. WHEN o job excede as tentativas THEN SHALL ir para a fila `discarded` e logar

**Independent Test**: Inserir um job manualmente via código e verificar no banco que ele foi processado (`SELECT * FROM river_jobs WHERE state = 'completed'`).

---

### P1: Cron insere jobs em vez de executar diretamente ⭐ MVP

**User Story**: Como desenvolvedor, quero que o cron agende jobs no River em vez de executar a lógica diretamente, para demonstrar o padrão scheduler/worker.

**Acceptance Criteria**:

1. WHEN a rotina é disparada pelo cron THEN SHALL inserir N jobs (um por destinatário) no River, em lote
2. WHEN os jobs são inseridos THEN o cron SHALL retornar imediatamente (não aguarda execução)
3. WHEN os jobs são inseridos THEN SHALL logar `"N jobs enqueued for routine X"`

**Independent Test**: Acionar o cron e verificar com `SELECT count(*) FROM river_jobs WHERE state = 'available'` que os jobs foram criados.

---

### P1: River worker processa jobs ⭐ MVP

**User Story**: Como desenvolvedor, quero que um River worker rode no mesmo processo e processe os jobs em paralelo, para demonstrar workers concorrentes.

**Acceptance Criteria**:

1. WHEN o worker (River) inicia THEN SHALL processar jobs da fila `default` com concorrência configurável
2. WHEN um job é processado com sucesso THEN state SHALL mudar para `completed`
3. WHEN um job falha THEN state SHALL mudar para `retryable` e ser reagendado

**Independent Test**: Após jobs inseridos, verificar `SELECT state, count(*) FROM river_jobs GROUP BY state` mostrando `completed`.

---

## Requirement Traceability

| ID | Story | Status |
|---|---|---|
| RIVER-01 | P1: PostgreSQL no docker-compose | Pending |
| RIVER-02 | P1: Job SendWhatsAppMessageJob | Pending |
| RIVER-03 | P1: Cron insere jobs | Pending |
| RIVER-04 | P1: River worker processa jobs | Pending |

## Success Criteria

- [ ] `docker compose up` sobe PostgreSQL e o worker conecta sem erros
- [ ] Tabelas `river_*` existem no banco após startup
- [ ] Cron insere jobs no banco e retorna imediatamente
- [ ] River worker processa jobs e `river_jobs` mostra registros `completed`
- [ ] Retry automático funciona: job que falha volta para `retryable`
