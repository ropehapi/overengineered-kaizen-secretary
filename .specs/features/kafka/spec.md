# Kafka — Especificação

## Problem Statement

Atualmente, a rotina chama o messaging-officer via HTTP de forma síncrona: se o messaging-officer estiver down, a mensagem é perdida. Para fins educacionais, queremos demonstrar o padrão pub/sub com Kafka, onde o scheduler publica eventos em um topic e um consumer independente lê e entrega as mensagens — desacoplando temporalmente o produtor do consumidor.

## Goals

- [ ] Rotina publica eventos de mensagem em topic Kafka em vez de chamar HTTP diretamente
- [ ] Consumer separado lê do topic e chama o messaging-officer
- [ ] Kafka + Zookeeper (ou KRaft) disponíveis via docker-compose
- [ ] Demonstrar desacoplamento: producer e consumer podem estar down em momentos diferentes

## Out of Scope

| Feature | Razão |
|---|---|
| Consumer group com múltiplos consumers | Adiciona complexidade sem valor didático proporcional |
| Schema Registry / Avro | Overkill para uma demo; JSON é suficiente |
| Kafka UI / AKHQ | Pode ser adicionado como bônus, não é requisito |
| Retenção e compactação de topics | Configuração padrão é suficiente para demo |

---

## User Stories

### P1: Kafka no docker-compose ⭐ MVP

**User Story**: Como desenvolvedor, quero Kafka disponível via docker-compose, para que a demo seja reproduzível sem infraestrutura externa.

**Acceptance Criteria**:

1. WHEN `docker compose up` é executado THEN Kafka SHALL estar disponível em `localhost:9092`
2. WHEN o Kafka sobe THEN o topic `whatsapp.messages.pending` SHALL ser criado automaticamente
3. WHEN o worker sobe THEN SHALL conectar ao Kafka usando `KAFKA_BROKERS` do ambiente

**Independent Test**: `kafka-topics.sh --list --bootstrap-server localhost:9092` lista o topic `whatsapp.messages.pending`.

---

### P1: Producer — rotina publica eventos no topic ⭐ MVP

**User Story**: Como desenvolvedor, quero que a rotina publique um evento Kafka por destinatário em vez de chamar o messaging-officer diretamente, para demonstrar desacoplamento via mensageria.

**Acceptance Criteria**:

1. WHEN a rotina é disparada THEN SHALL publicar N mensagens no topic `whatsapp.messages.pending` (uma por destinatário)
2. WHEN a mensagem é publicada THEN o payload SHALL ser JSON com campos `recipient_phone`, `recipient_name` e `message`
3. WHEN a publicação falha THEN SHALL logar o erro e continuar para o próximo destinatário
4. WHEN todos os eventos são publicados THEN SHALL logar `"N events published to Kafka"`

**Independent Test**: Acionar a rotina e verificar com `kafka-console-consumer.sh` que os eventos aparecem no topic.

---

### P1: Consumer — lê topic e chama messaging-officer ⭐ MVP

**User Story**: Como desenvolvedor, quero um consumer Kafka que leia o topic e chame o messaging-officer para cada mensagem, demonstrando o outro lado do canal assíncrono.

**Acceptance Criteria**:

1. WHEN o worker inicia THEN o consumer SHALL se inscrever no topic `whatsapp.messages.pending`
2. WHEN um evento é consumido THEN SHALL fazer POST para o messaging-officer com o payload da mensagem
3. WHEN o messaging-officer retorna sucesso THEN o consumer SHALL commitar o offset
4. WHEN o messaging-officer retorna erro THEN o consumer SHALL logar o erro e NÃO commitar (mensagem será reprocessada)
5. WHEN o consumer recebe sinal de shutdown THEN SHALL fechar graciosamente (commit final + close)

**Independent Test**: Consumer rodando + publicar mensagem manualmente no topic → verificar que o messaging-officer foi chamado.

---

### P2: Kafka UI no docker-compose

**User Story**: Como desenvolvedor, quero uma UI para visualizar topics e mensagens no Kafka, para tornar a demo mais visual durante a apresentação.

**Acceptance Criteria**:

1. WHEN `docker compose up` é executado THEN Kafka UI SHALL estar disponível em `localhost:8080`
2. WHEN a UI abre THEN SHALL exibir o topic `whatsapp.messages.pending` com as mensagens publicadas

**Independent Test**: Abrir `localhost:8080`, navegar até o topic e ver as mensagens publicadas pela rotina.

---

## Requirement Traceability

| ID | Story | Status |
|---|---|---|
| KAFKA-01 | P1: Kafka no docker-compose | Pending |
| KAFKA-02 | P1: Producer publica eventos | Pending |
| KAFKA-03 | P1: Consumer lê e entrega | Pending |
| KAFKA-04 | P2: Kafka UI no docker-compose | Pending |

## Success Criteria

- [ ] Kafka sobe com `docker compose up` e topic é criado automaticamente
- [ ] Rotina publica eventos e retorna imediatamente (sem aguardar entrega)
- [ ] Consumer lê eventos e chama o messaging-officer
- [ ] Kafka UI mostra mensagens no topic em tempo real
- [ ] Se consumer está down, mensagens ficam no topic aguardando (durabilidade)
