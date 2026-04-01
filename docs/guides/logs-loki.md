# Logs — Loki + Promtail

## Como funciona

```
app (stderr) → Docker → Promtail → Loki → Grafana
```

- **Zap**: biblioteca de logging da Uber. Escreve logs estruturados em JSON no stderr do processo. Substitui o `slog` da stdlib — diferencial é a API com campos tipados (`zap.String`, `zap.Error`, `zap.Int`) e zero alocações no hot path.
- **Promtail**: agente que descobre containers via Docker socket e coleta seus logs em tempo real, enviando ao Loki.
- **Loki**: armazena e indexa os logs por labels (ex: `container`, `service`). Não indexa o conteúdo — apenas os labels — o que o torna leve comparado ao Elasticsearch.
- **Grafana**: consome o Loki como datasource e permite explorar/filtrar os logs.

## Passo a passo para demonstrar

1. Subir o projeto
```sh
sudo docker compose up -d
go run cmd/worker/main.go
```

2. Abrir o [Grafana](http://localhost:3001) → **Explore** → selecionar datasource **Loki**

3. Inserir a query e executar:
```
{container="kaizen-secretary"}
```

4. Os logs aparecem em tempo real. Para filtrar por nível:
```
{container="kaizen-secretary"} | json | level="error"
```

## Modos de output (Zap)

| `APP_ENV` | Formato | Uso |
|---|---|---|
| não definido | legível, colorido | desenvolvimento local |
| `production` | JSON compacto | produção / coleta por Promtail |

```sh
# development (padrão)
go run cmd/worker/main.go

# production
APP_ENV=production go run cmd/worker/main.go
```
