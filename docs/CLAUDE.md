# Kaizen Secretary

## Visão geral

Worker Go baseado em cronjobs que executa rotinas agendadas automaticamente. Atualmente é usado para enviar lembretes de mensalidade de escoteiros via WhatsApp, mas é projetado para ser extensível com qualquer rotina dentro de `internal/routines/`.

## Ecossistema

Este projeto faz parte do ecossistema **"manda-pra-mim"** de automação WhatsApp:
- **messaging-officer** → API REST que conecta ao WhatsApp via Baileys (porta 3000). Este worker consome essa API para enviar mensagens.
- **kaizen-wpp-scheduler-backend** → API REST em Go para gerenciamento de agendamentos (porta 8080).
- **kaizen-wpp-scheduler-frontend** → Frontend React para interface de gerenciamento de agendamentos.

## Stack e dependências

- **Go 1.24.2**
- **robfig/cron/v3** → Biblioteca de cronjobs com suporte a segundos
- **joho/godotenv** → Carregamento de variáveis de ambiente via `.env`

## Arquitetura

```
cmd/worker/main.go              # Entry point — configura e inicia os cron jobs
internal/routines/               # Pasta para rotinas agendadas (uma por arquivo)
  mensalidadesEscoteiro.go       # Rotina de lembrete de mensalidades escoteiras
```

### Padrão de execução

1. Carrega `.env` com `godotenv`
2. Cria um cron scheduler com suporte a segundos (`cron.WithSeconds()`)
3. Registra funções de rotina com expressões cron
4. Inicia o scheduler e bloqueia com `select{}` para manter o processo vivo

### Formato de expressão cron (6 campos)

```
"Segundo Minuto Hora DiaMês Mês DiaSemana"
Exemplo: "0 0 0 10 * *" = meia-noite do dia 10 de cada mês
```

## Variáveis de ambiente

| Variável | Descrição | Exemplo |
|---|---|---|
| `MESSAGING_OFFICER_HOST` | Host da API messaging-officer | `http://localhost` |
| `MESSAGING_OFFICER_PORT` | Porta da API messaging-officer | `3000` |

## Como adicionar novas rotinas

1. Crie um novo arquivo `.go` em `internal/routines/`
2. Defina uma função pública (ex: `func MinhaNovaRotina()`)
3. Registre a função no `cmd/worker/main.go` com `c.AddFunc("expressão_cron", routines.MinhaNovaRotina)`

## Padrão de comunicação com APIs

As rotinas fazem chamadas HTTP diretamente usando `net/http`. Veja `mensalidadesEscoteiro.go` como referência:
- Monta payload JSON com `json.Marshal`
- Faz POST para o endpoint do messaging-officer
- Faz parse e log da resposta

## Convenções

- Rotinas devem ser funções `func()` sem parâmetros (exigência do cron)
- Configurações externas devem vir de variáveis de ambiente (`os.Getenv`)
- Logs usam `fmt.Println` (não há logger estruturado neste projeto)
- Nomes de arquivos de rotinas devem ser descritivos em português ou inglês

## Como executar

```bash
# Desenvolvimento local
go run cmd/worker/main.go

# Build
go build -o bin/worker ./cmd/worker
```
