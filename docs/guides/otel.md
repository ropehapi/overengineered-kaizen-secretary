# Otel + Jaeger
Passo a passo para testar/demonstrar a implementação:

1. Subir o projeto
```sh
sudo docker compose up -d
go run cmd/worker/main.go
```

2. Verificar os traces no [painel do Jaeger](http://localhost:16686).