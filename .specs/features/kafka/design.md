# Kafka — Design

**Spec**: `.specs/features/kafka/spec.md`
**Status**: Draft

---

## Architecture Overview

```
ANTES:
  cron → rotina → HTTP POST → messaging-officer

DEPOIS:
  cron → rotina (producer) → [Kafka: whatsapp.messages.pending]
                                          │
                              consumer (goroutine no mesmo processo)
                                          │
                                    HTTP POST
                                          │
                                  messaging-officer
```

**Dois papéis no mesmo processo** (para simplificar a demo):
- **Producer**: rotina cron publica eventos no topic
- **Consumer**: goroutine lê do topic e chama o messaging-officer

Em produção real, producer e consumer seriam serviços separados — esse é o ponto de discussão na apresentação.

```
cmd/worker/main.go
  ├── kafka.InitProducer()          ← producer síncrono
  ├── kafka.InitConsumer()          ← consumer assíncrono (goroutine)
  │     └── go consumer.ReadLoop(ctx)
  │
  ├── c.AddFunc(expr, func() {
  │       routines.PublishScoutMonthlyFees(ctx, producer)
  │   })
  │
  └── select {}
```

---

## Code Reuse Analysis

### Existing Components to Leverage

| Component | Location | How to Use |
|---|---|---|
| Map de contribuintes | `internal/routines/mensalidadesEscoteiro.go` | Exportar como `Contributors()` ou mover para `internal/data/` |
| Lógica de HTTP POST | `mensalidadesEscoteiro.go` | Mover para consumer (`internal/kafka/consumer.go`) |
| Lógica de construção de mensagem | `mensalidadesEscoteiro.go` | Extrair `BuildMessage(name string) string` |

### Integration Points

| Sistema | Método |
|---|---|
| Kafka | `github.com/segmentio/kafka-go` — producer síncrono + consumer com ReadMessage |
| `docker-compose.yml` | Service Kafka (KRaft mode — sem Zookeeper) + Kafka UI |
| `cmd/worker/main.go` | Inicializar producer e consumer, passar producer para rotina |

---

## Components

### `internal/kafka/producer.go`

- **Purpose**: Encapsular o Kafka producer com método para publicar evento de mensagem WhatsApp
- **Location**: `internal/kafka/producer.go` (novo arquivo)
- **Interfaces**:
  - `NewProducer(brokers []string) *Producer`
  - `Publish(ctx context.Context, msg WhatsAppMessageEvent) error`
  - `Close() error`
- **Payload (JSON)**:
  ```go
  type WhatsAppMessageEvent struct {
      RecipientPhone string `json:"recipient_phone"`
      RecipientName  string `json:"recipient_name"`
      Message        string `json:"message"`
  }
  ```
- **Dependencies**: `github.com/segmentio/kafka-go`

### `internal/kafka/consumer.go`

- **Purpose**: Ler eventos do topic e chamar o messaging-officer para cada um
- **Location**: `internal/kafka/consumer.go` (novo arquivo)
- **Interfaces**:
  - `NewConsumer(brokers []string, topic, groupID string) *Consumer`
  - `ReadLoop(ctx context.Context)` — bloqueia até ctx cancelado
  - `Close() error`
- **Lógica de ReadLoop**:
  ```go
  for {
      msg, err := reader.ReadMessage(ctx)  // bloqueia até chegar mensagem
      if err != nil { break }              // ctx cancelado → sai
      var event WhatsAppMessageEvent
      json.Unmarshal(msg.Value, &event)
      if err := sendToMessagingOfficer(event); err != nil {
          // log erro mas NÃO commita → mensagem será reprocessada
          continue
      }
      // commit implícito com ReadMessage (auto-commit por padrão no kafka-go)
  }
  ```

### Modificação em `internal/routines/mensalidadesEscoteiro.go`

- **O que muda**: Função passa a publicar eventos no Kafka em vez de chamar HTTP:
  ```go
  func PublishScoutMonthlyFees(ctx context.Context, producer *kafka.Producer) error {
      for name, phone := range contributors {
          event := kafka.WhatsAppMessageEvent{
              RecipientPhone: phone,
              RecipientName:  name,
              Message:        buildMessage(name),
          }
          if err := producer.Publish(ctx, event); err != nil {
              zap.L().Error("failed to publish event", zap.String("recipient", phone), zap.Error(err))
          }
      }
      return nil
  }
  ```

### `docker-compose.yml` — Kafka (KRaft) + Kafka UI

- **Kafka image**: `confluentinc/cp-kafka` em modo KRaft (sem Zookeeper)
- **Kafka UI**: `provectuslabs/kafka-ui`

---

## Tech Decisions

| Decisão | Escolha | Rationale |
|---|---|---|
| Kafka client library | `segmentio/kafka-go` | API idiomática Go, sem dependência de librdkafka |
| Kafka mode | KRaft (sem Zookeeper) | Zookeeper é deprecated; KRaft é mais simples para demo |
| Consumer no mesmo processo | Sim (goroutine) | Simplifica demo; em prod seria serviço separado — ponto de discussão |
| Topic creation | Auto-create no Kafka config | `KAFKA_AUTO_CREATE_TOPICS_ENABLE=true` |
| Serialização | JSON | Simples e legível; Avro/Protobuf seriam overkill para demo |

---

## Dependencies a adicionar

```
github.com/segmentio/kafka-go
```

## Variáveis de ambiente a adicionar ao `.env.example`

```
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC=whatsapp.messages.pending
```
