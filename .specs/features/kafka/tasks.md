# Kafka — Tasks

**Design**: `.specs/features/kafka/design.md`
**Status**: Draft

---

## Execution Plan

```
Phase 1 (Sequential — Foundation):
  T1 → T2

Phase 2 (Parallel — Producer e Consumer):
  T2 complete, então:
    ├── T3 [P]  (producer)
    ├── T4 [P]  (consumer)
    └── T5 [P]  (Kafka no docker-compose)

Phase 3 (Sequential — Rotina e Integração):
  T3 complete, então:
    T6

Phase 4 (Sequential — Wiring e Validação):
  T4 + T6 + T5 completos, então:
    T7 → T8
```

---

## Task Breakdown

### T1: Adicionar dependência `kafka-go` ao go.mod

**What**: Rodar `go get github.com/segmentio/kafka-go` para adicionar o client Kafka.
**Where**: `go.mod` / `go.sum`
**Depends on**: Nenhuma
**Requirement**: KAFKA-01

**Done when**:
- [ ] `go.mod` contém `github.com/segmentio/kafka-go`
- [ ] `go build ./...` compila sem erros

**Verify**:
```bash
go build ./...
```

**Commit**: `chore: add segmentio/kafka-go dependency`

---

### T2: Criar `internal/kafka/types.go` — evento de mensagem WhatsApp

**What**: Definir a struct `WhatsAppMessageEvent` usada como payload JSON no topic.
**Where**: `internal/kafka/types.go` (novo arquivo)
**Depends on**: T1
**Requirement**: KAFKA-02

**Implementação**:
```go
package kafka

type WhatsAppMessageEvent struct {
    RecipientPhone string `json:"recipient_phone"`
    RecipientName  string `json:"recipient_name"`
    Message        string `json:"message"`
}
```

**Done when**:
- [ ] Arquivo `internal/kafka/types.go` criado
- [ ] Struct com 3 campos JSON exportados
- [ ] `go build ./...` compila sem erros

**Verify**:
```bash
go build ./...
```

**Commit**: `feat(kafka): define WhatsAppMessageEvent payload type`

---

### T3: Criar `internal/kafka/producer.go` [P]

**What**: Implementar producer que publica eventos JSON no topic `whatsapp.messages.pending`.
**Where**: `internal/kafka/producer.go` (novo arquivo)
**Depends on**: T2
**Requirement**: KAFKA-02

**Implementação**:
```go
package kafka

import (
    "context"
    "encoding/json"
    "github.com/segmentio/kafka-go"
)

type Producer struct {
    writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
    return &Producer{
        writer: &kafka.Writer{
            Addr:     kafka.TCP(brokers...),
            Topic:    topic,
            Balancer: &kafka.LeastBytes{},
        },
    }
}

func (p *Producer) Publish(ctx context.Context, event WhatsAppMessageEvent) error {
    payload, err := json.Marshal(event)
    if err != nil {
        return fmt.Errorf("failed to marshal event: %w", err)
    }
    return p.writer.WriteMessages(ctx, kafka.Message{
        Key:   []byte(event.RecipientPhone),
        Value: payload,
    })
}

func (p *Producer) Close() error {
    return p.writer.Close()
}
```

**Done when**:
- [ ] `NewProducer(brokers, topic)` retorna `*Producer`
- [ ] `Publish()` serializa evento em JSON e escreve no topic
- [ ] `Key` da mensagem é o `RecipientPhone` (garante ordering por destinatário)
- [ ] `Close()` exposto para graceful shutdown
- [ ] `go build ./...` compila sem erros

**Verify**:
```bash
go build ./...
go vet ./internal/kafka/...
```

**Commit**: `feat(kafka): implement Kafka producer for WhatsApp message events`

---

### T4: Criar `internal/kafka/consumer.go` [P]

**What**: Implementar consumer que lê do topic e chama o messaging-officer via HTTP para cada evento.
**Where**: `internal/kafka/consumer.go` (novo arquivo)
**Depends on**: T2
**Reuses**: Lógica de HTTP POST de `internal/routines/mensalidadesEscoteiro.go`
**Requirement**: KAFKA-03

**Implementação**:
```go
package kafka

import (
    "context"
    "encoding/json"
    "os"
    "github.com/segmentio/kafka-go"
    "go.uber.org/zap"
)

type Consumer struct {
    reader *kafka.Reader
}

func NewConsumer(brokers []string, topic, groupID string) *Consumer {
    return &Consumer{
        reader: kafka.NewReader(kafka.ReaderConfig{
            Brokers: brokers,
            Topic:   topic,
            GroupID: groupID,
        }),
    }
}

func (c *Consumer) ReadLoop(ctx context.Context) {
    for {
        msg, err := c.reader.ReadMessage(ctx)
        if err != nil {
            if ctx.Err() != nil { return }  // ctx cancelado = shutdown normal
            zap.L().Error("kafka read error", zap.Error(err))
            continue
        }

        var event WhatsAppMessageEvent
        if err := json.Unmarshal(msg.Value, &event); err != nil {
            zap.L().Error("failed to unmarshal kafka message", zap.Error(err))
            continue
        }

        if err := sendToMessagingOfficer(event); err != nil {
            zap.L().Error("failed to deliver message",
                zap.String("recipient", event.RecipientPhone),
                zap.Error(err))
            continue
        }

        zap.L().Info("message delivered via kafka consumer",
            zap.String("recipient", event.RecipientPhone))
    }
}

func (c *Consumer) Close() error { return c.reader.Close() }

func sendToMessagingOfficer(event WhatsAppMessageEvent) error {
    // HTTP POST para MESSAGING_OFFICER_HOST:MESSAGING_OFFICER_PORT
    // (extraído de mensalidadesEscoteiro.go)
}
```

**Done when**:
- [ ] `NewConsumer(brokers, topic, groupID)` retorna `*Consumer`
- [ ] `ReadLoop(ctx)` bloqueia até `ctx` cancelado
- [ ] Erro de parse: loga e continua (não trava o loop)
- [ ] Erro de entrega: loga e continua (kafka-go faz auto-commit, offset não avança em erro severo)
- [ ] `sendToMessagingOfficer()` privada, lê host/porta do ambiente
- [ ] `go build ./...` compila sem erros

**Verify**:
```bash
go build ./...
go vet ./internal/kafka/...
```

**Commit**: `feat(kafka): implement Kafka consumer that delivers to messaging-officer`

---

### T5: Adicionar Kafka (KRaft) e Kafka UI ao `docker-compose.yml` [P]

**What**: Adicionar services `kafka` em modo KRaft (sem Zookeeper) e `kafka-ui` para visualização.
**Where**: `docker-compose.yml` (modificação)
**Depends on**: T1
**Requirement**: KAFKA-01, KAFKA-04

**Services a adicionar**:
```yaml
kafka:
  image: confluentinc/cp-kafka:7.6.0
  container_name: kaizen-kafka
  ports:
    - "9092:9092"
  environment:
    KAFKA_NODE_ID: 1
    KAFKA_PROCESS_ROLES: broker,controller
    KAFKA_LISTENERS: PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:9093
    KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092
    KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: PLAINTEXT:PLAINTEXT,CONTROLLER:PLAINTEXT
    KAFKA_CONTROLLER_QUORUM_VOTERS: 1@kafka:9093
    KAFKA_CONTROLLER_LISTENER_NAMES: CONTROLLER
    KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
    KAFKA_AUTO_CREATE_TOPICS_ENABLE: "true"
    CLUSTER_ID: "MkU3OEVBNTcwNTJENDM2Qg"
  networks:
    - manda-pra-mim

kafka-ui:
  image: provectuslabs/kafka-ui:latest
  container_name: kaizen-kafka-ui
  ports:
    - "8080:8080"
  environment:
    - KAFKA_CLUSTERS_0_NAME=local
    - KAFKA_CLUSTERS_0_BOOTSTRAPSERVERS=kafka:9092
  depends_on:
    - kafka
  networks:
    - manda-pra-mim
```

**Variáveis a adicionar no service `app`**:
```yaml
environment:
  - KAFKA_BROKERS=kafka:9092
  - KAFKA_TOPIC=whatsapp.messages.pending
```

**Atualizar `.env.example`**:
```
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC=whatsapp.messages.pending
```

**Done when**:
- [ ] Service `kafka` em modo KRaft adicionado (sem Zookeeper)
- [ ] Service `kafka-ui` adicionado na porta `8080`
- [ ] `AUTO_CREATE_TOPICS_ENABLE=true` configurado
- [ ] Variáveis `KAFKA_BROKERS` e `KAFKA_TOPIC` no service `app`
- [ ] `docker compose config` valida sem erros

**Verify**:
```bash
docker compose config
docker compose up kafka -d
sleep 15  # Kafka precisa de tempo para inicializar
docker compose exec kafka kafka-topics --list --bootstrap-server localhost:9092
docker compose down
```

**Commit**: `feat(kafka): add Kafka KRaft and Kafka UI to docker-compose`

---

### T6: Refatorar `mensalidadesEscoteiro.go` para publicar no Kafka

**What**: Renomear/refatorar a função principal para `PublishScoutMonthlyFees` que usa o producer Kafka.
**Where**: `internal/routines/mensalidadesEscoteiro.go` (modificação)
**Depends on**: T3
**Requirement**: KAFKA-02

**O que muda**:
```go
// ANTES: executa envio HTTP diretamente
func RememberScoutMonthlyFees() { /* HTTP loop */ }

// DEPOIS: publica eventos no Kafka
func PublishScoutMonthlyFees(ctx context.Context, producer *kafka.Producer) error {
    for name, phone := range contributors {
        event := kafka.WhatsAppMessageEvent{
            RecipientPhone: phone,
            RecipientName:  name,
            Message:        buildMessage(name),
        }
        if err := producer.Publish(ctx, event); err != nil {
            zap.L().Error("failed to publish to kafka",
                zap.String("recipient", phone),
                zap.Error(err))
            // continua para próximo destinatário
        }
    }
    zap.L().Info("events published to kafka",
        zap.Int("count", len(contributors)),
        zap.String("topic", os.Getenv("KAFKA_TOPIC")))
    return nil
}
```

**Done when**:
- [ ] `PublishScoutMonthlyFees(ctx, producer)` substitui `RememberScoutMonthlyFees()`
- [ ] Nenhuma chamada HTTP ao messaging-officer neste arquivo
- [ ] Log de quantidade de eventos publicados ao final
- [ ] `go build ./...` compila sem erros

**Verify**:
```bash
go build ./...
grep -n "http\." internal/routines/mensalidadesEscoteiro.go || echo "OK - sem HTTP na rotina"
```

**Commit**: `feat(kafka): refactor scout routine to publish Kafka events instead of direct HTTP`

---

### T7: Integrar producer e consumer em `cmd/worker/main.go`

**What**: Inicializar producer e consumer Kafka no startup, passar producer para a rotina, iniciar consumer em goroutine.
**Where**: `cmd/worker/main.go` (modificação)
**Depends on**: T4, T6
**Requirement**: KAFKA-01, KAFKA-03

**O que adicionar**:
```go
brokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
topic := os.Getenv("KAFKA_TOPIC")

producer := kafka.NewProducer(brokers, topic)
defer producer.Close()

consumer := kafka.NewConsumer(brokers, topic, "kaizen-secretary")
defer consumer.Close()

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go consumer.ReadLoop(ctx)

c.AddFunc(expr, func() {
    if err := routines.PublishScoutMonthlyFees(ctx, producer); err != nil {
        zap.L().Error("failed to publish routine events", zap.Error(err))
    }
})
```

**Done when**:
- [ ] `KAFKA_BROKERS` e `KAFKA_TOPIC` lidos do ambiente
- [ ] Producer inicializado com `defer Close()`
- [ ] Consumer inicializado com `defer Close()`
- [ ] `go consumer.ReadLoop(ctx)` iniciado em goroutine
- [ ] Context com cancel para graceful shutdown do consumer
- [ ] `AddFunc` usa `routines.PublishScoutMonthlyFees`
- [ ] `go build ./cmd/worker` compila sem erros

**Verify**:
```bash
go build ./cmd/worker
```

**Commit**: `feat(kafka): wire Kafka producer and consumer in worker main`

---

### T8: Validação E2E — eventos no Kafka UI e entrega ao messaging-officer

**What**: Subir o stack completo e verificar o fluxo: cron → Kafka topic → consumer → messaging-officer.
**Where**: docker-compose stack + Kafka UI
**Depends on**: T5, T7
**Requirement**: KAFKA-01, KAFKA-02, KAFKA-03, KAFKA-04

**Passos de validação**:

1. **Teste 1 — Producer**:
   - `docker compose up --build`
   - Alterar cron para disparar em 10s temporariamente
   - Acessar Kafka UI em `http://localhost:8080`
   - Verificar topic `whatsapp.messages.pending` com mensagens
   - Inspecionar payload JSON de uma mensagem

2. **Teste 2 — Consumer**:
   - Verificar logs do worker: `"message delivered via kafka consumer"` por destinatário
   - Verificar que messaging-officer recebeu as chamadas (logs do serviço)

3. **Teste 3 — Durabilidade** (ponto didático da apresentação):
   - Parar o consumer (`Ctrl+C` no worker)
   - Publicar eventos manualmente:
     ```bash
     docker compose exec kafka kafka-console-producer \
       --topic whatsapp.messages.pending \
       --bootstrap-server localhost:9092
     ```
   - Reiniciar o worker
   - Verificar que as mensagens pendentes são processadas (consumer group offset)

4. Reverter cron para expressão original

**Done when**:
- [ ] Kafka UI mostra topic com mensagens após disparo do cron
- [ ] Payload JSON das mensagens tem `recipient_phone`, `recipient_name`, `message`
- [ ] Logs do worker mostram entrega de cada mensagem pelo consumer
- [ ] Teste de durabilidade: mensagens publicadas com consumer down são entregues ao reiniciar
- [ ] Cron revertido para expressão original

**Commit**: `test(kafka): validate end-to-end Kafka producer/consumer flow`

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
  T3 complete, então:
    T6

Phase 4 (Sequential):
  T4 + T6 + T5 completos, então:
    T7 ──→ T8
```

---

## Task Granularity Check

| Task | Escopo | Status |
|---|---|---|
| T1: Dependência kafka-go | `go.mod` | ✅ Granular |
| T2: types.go | 1 struct | ✅ Granular |
| T3: producer.go | 1 arquivo, 1 componente | ✅ Granular |
| T4: consumer.go | 1 arquivo, 1 componente | ✅ Granular |
| T5: Kafka + UI no compose | 2 services | ✅ Granular |
| T6: Refatorar rotina | 1 arquivo, publish | ✅ Granular |
| T7: Wiring no main | 1 arquivo, integração | ✅ Granular |
| T8: Validação E2E | Teste manual + durabilidade | ✅ Granular |
