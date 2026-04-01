# OpenTelemetry + Jaeger — Especificação

## Problem Statement

O worker executa rotinas críticas (envio de ~50 mensagens WhatsApp) sem nenhuma observabilidade de tracing. Quando uma rotina falha parcialmente, não há visibilidade de onde o tempo foi gasto, quais envios travaram, ou qual chamada HTTP gerou timeout. Para fins educacionais, queremos demonstrar tracing distribuído como forma de tornar o fluxo de execução visível e inspecionável.

## Goals

- [ ] Instrumentar o worker com OpenTelemetry para gerar traces de cada execução de rotina
- [ ] Visualizar os traces no Jaeger UI via docker-compose
- [ ] Demonstrar context propagation em chamadas HTTP para o messaging-officer

## Out of Scope

| Feature | Razão |
|---|---|
| Tracing de serviços externos (messaging-officer) | Exigiria instrumentar outro serviço fora deste repo |
| Sampling dinâmico | Desnecessário para fins educacionais |
| Persistência de traces além do Jaeger in-memory | Overhead de infra sem valor didático adicional |

---

## User Stories

### P1: Tracer inicializado no worker ⭐ MVP

**User Story**: Como desenvolvedor, quero que o worker inicialize um tracer OTLP ao subir, para que todos os traces sejam enviados ao Jaeger automaticamente.

**Acceptance Criteria**:

1. WHEN o worker inicia THEN o tracer SHALL ser configurado com exporter OTLP apontando para o Jaeger
2. WHEN o worker encerra THEN o tracer SHALL fazer flush dos spans pendentes (graceful shutdown)
3. WHEN a variável `OTEL_EXPORTER_OTLP_ENDPOINT` não está definida THEN o worker SHALL usar `http://localhost:4318` como default

**Independent Test**: Subir o stack via docker-compose e verificar que o Jaeger UI (`localhost:16686`) exibe o serviço `kaizen-secretary`.

---

### P1: Span por execução de rotina ⭐ MVP

**User Story**: Como desenvolvedor, quero que cada execução de rotina cron gere um trace raiz, para que eu possa ver quando e quanto tempo cada rotina levou.

**Acceptance Criteria**:

1. WHEN uma rotina é disparada pelo cron THEN um span raiz SHALL ser criado com o nome da rotina
2. WHEN a rotina termina (sucesso ou falha) THEN o span SHALL ser encerrado com o status correto
3. WHEN a rotina falha THEN o span SHALL registrar o erro como atributo do span

**Independent Test**: Acionar a rotina manualmente (alterar cron para rodar em 10s) e verificar no Jaeger o trace `RememberScoutMonthlyFees`.

---

### P1: Span por envio de mensagem ⭐ MVP

**User Story**: Como desenvolvedor, quero que cada chamada HTTP para o messaging-officer seja um span filho, para que eu possa ver qual envio específico foi lento ou falhou.

**Acceptance Criteria**:

1. WHEN a rotina itera sobre os destinatários THEN cada chamada HTTP SHALL gerar um span filho com atributo `recipient.phone`
2. WHEN o HTTP retorna status >= 400 THEN o span SHALL ser marcado com erro e o status code como atributo
3. WHEN o HTTP retorna sucesso THEN o span SHALL registrar a duração da chamada

**Independent Test**: Abrir um trace no Jaeger e verificar spans filhos com nome `send_whatsapp_message`, cada um com atributo `recipient.phone`.

---

### P2: Jaeger no docker-compose

**User Story**: Como desenvolvedor, quero subir o Jaeger com um único `docker compose up`, para que a demonstração seja reproduzível sem configuração manual.

**Acceptance Criteria**:

1. WHEN `docker compose up` é executado THEN o Jaeger SHALL estar disponível em `localhost:16686`
2. WHEN o worker sobe THEN SHALL se conectar automaticamente ao Jaeger via OTLP HTTP (porta 4318)

**Independent Test**: `curl http://localhost:16686` retorna 200.

---

## Requirement Traceability

| ID | Story | Status |
|---|---|---|
| OTEL-01 | P1: Tracer inicializado | Pending |
| OTEL-02 | P1: Span por rotina | Pending |
| OTEL-03 | P1: Span por mensagem | Pending |
| OTEL-04 | P2: Jaeger no docker-compose | Pending |

## Success Criteria

- [ ] Jaeger UI exibe traces com hierarquia: rotina → mensagens individuais
- [ ] Cada span filho tem atributos `recipient.phone` e status HTTP
- [ ] Stack completo sobe com `docker compose up`
