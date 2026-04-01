# Kafka
Passo a passo para testar/demonstrar a implementação:

1. Subir o projeto
```sh
sudo docker compose up -d
go run cmd/worker/main.go
```

2. Verificar a publicação de mensagens no tópico `whatsapp.messages.pending` no [Kafka UI](http://127.0.0.1:8080/)