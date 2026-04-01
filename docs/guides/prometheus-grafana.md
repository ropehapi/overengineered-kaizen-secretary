# Prometheu + Grafana
Passo a passo para testar/demonstrar a implementação:

1. Subir o projeto
```sh
sudo docker compose up -d
go run cmd/worker/main.go
```

2. Verificar as métricas no [painel do Grafana](http://localhost:3001).