# Prometheus + Grafana — Especificação

## Problem Statement

O worker executa rotinas automaticamente, mas não há visibilidade sobre quantas mensagens foram enviadas, quantas falharam, ou quanto tempo cada rotina levou ao longo do tempo. Para fins educacionais, queremos adicionar métricas expostas via endpoint HTTP que o Prometheus pode scrape e o Grafana pode visualizar em dashboards.

## Goals

- [ ] Expor métricas do worker em `/metrics` via HTTP (padrão Prometheus)
- [ ] Coletar métricas de execução de rotinas: total de execuções, duração, mensagens enviadas e falhas
- [ ] Visualizar métricas em dashboard Grafana pré-configurado
- [ ] Todo o stack sobe com um único `docker compose up`

## Out of Scope

| Feature | Razão |
|---|---|
| Alertas no Alertmanager | Adiciona complexidade sem valor didático adicional |
| Métricas de runtime Go (goroutines, GC) | Foco nas métricas de negócio, não de runtime |
| Persistência de dados do Prometheus | In-memory é suficiente para demo |

---

## User Stories

### P1: Endpoint `/metrics` exposto ⭐ MVP

**User Story**: Como desenvolvedor, quero que o worker exponha um endpoint HTTP `/metrics` no formato Prometheus, para que o Prometheus possa fazer scrape das métricas.

**Acceptance Criteria**:

1. WHEN o worker inicia THEN um servidor HTTP SHALL subir em paralelo na porta `9090` (configurável via env)
2. WHEN `GET /metrics` é chamado THEN SHALL retornar métricas no formato texto do Prometheus
3. WHEN o servidor HTTP falha ao iniciar THEN o worker SHALL logar o erro e encerrar

**Independent Test**: `curl http://localhost:9090/metrics` retorna texto com métricas no formato `# HELP / # TYPE`.

---

### P1: Métricas de execução de rotinas ⭐ MVP

**User Story**: Como desenvolvedor, quero métricas que registrem cada execução de rotina, para que eu possa ver frequência e duração no Grafana.

**Acceptance Criteria**:

1. WHEN uma rotina é executada THEN o counter `routine_executions_total{routine="nome"}` SHALL ser incrementado
2. WHEN uma rotina termina THEN o histogram `routine_duration_seconds{routine="nome"}` SHALL registrar a duração
3. WHEN uma rotina falha com panic ou erro THEN o counter `routine_errors_total{routine="nome"}` SHALL ser incrementado

**Independent Test**: Rodar a rotina e confirmar que `routine_executions_total` incrementou via `curl localhost:9090/metrics | grep routine_executions`.

---

### P1: Métricas de envio de mensagens ⭐ MVP

**User Story**: Como desenvolvedor, quero métricas granulares de cada mensagem enviada, para que eu possa ver taxa de sucesso e falha por rotina.

**Acceptance Criteria**:

1. WHEN uma mensagem é enviada com sucesso THEN `messages_sent_total{routine="nome", status="success"}` SHALL ser incrementado
2. WHEN uma mensagem falha THEN `messages_sent_total{routine="nome", status="failure"}` SHALL ser incrementado
3. WHEN a rotina termina THEN o gauge `messages_pending` SHALL refletir quantas mensagens ainda não foram processadas

**Independent Test**: Verificar `messages_sent_total` via curl após execução de rotina.

---

### P2: Prometheus + Grafana no docker-compose

**User Story**: Como desenvolvedor, quero que Prometheus e Grafana estejam no docker-compose com configuração pronta, para que a demo seja reproduzível.

**Acceptance Criteria**:

1. WHEN `docker compose up` é executado THEN Prometheus SHALL estar disponível em `localhost:9091`
2. WHEN `docker compose up` é executado THEN Grafana SHALL estar disponível em `localhost:3001`
3. WHEN Grafana sobe THEN a datasource do Prometheus SHALL estar pré-configurada (sem configuração manual)
4. WHEN Grafana sobe THEN um dashboard básico de kaizen-secretary SHALL estar disponível

**Independent Test**: Acessar `localhost:3001`, fazer login (admin/admin), e encontrar dashboard "Kaizen Secretary" com os panels de métricas.

---

## Requirement Traceability

| ID | Story | Status |
|---|---|---|
| PROM-01 | P1: Endpoint /metrics | Pending |
| PROM-02 | P1: Métricas de rotinas | Pending |
| PROM-03 | P1: Métricas de mensagens | Pending |
| PROM-04 | P2: Stack no docker-compose | Pending |

## Success Criteria

- [ ] `curl localhost:9090/metrics` retorna métricas formatadas
- [ ] Prometheus (`localhost:9091`) mostra target `kaizen-secretary` como UP
- [ ] Grafana (`localhost:3001`) exibe dashboard com métricas ao vivo
- [ ] Stack completo sobe com `docker compose up`
