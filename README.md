# Kaizen secretary
Esse aplicação é um worker baseado em cronjobs que dispara rotinas de qualquer outra coisa que você ousar desenvolver dentro da pasta `/internal/routines/`, como envio de mensagens automatizadas, por exemplo.

## Pré requisitos
- Go 1.24.2
- Qualquer coisa necessária para rodar as rotinas

## Instalação
1. Clone e acesse o diretório do repositório
```
git clone git@github.com:ropehapi/kaizen-secretary.git
cd kaizen-secretary/
```
2. Configure as variáveis de ambiente conforme o desejado no arquivo `.env`.
3. Execute a aplicação
```
go run cmd/worker/main.go
```

## Uso do cron
Exemplo do Cron "0 30 0 14 * *"

| Campo         | Valor | Significado                |
|---------------|-------|----------------------------|
| Segundo       | 0     | no segundo 0              |
| Minuto        | 30    | no minuto 30              |
| Hora          | 0     | à meia-noite (00 horas)   |
| Dia do mês    | 14    | no dia 14                 |
| Mês           | *     | todo mês                  |
| Dia da semana | *     | qualquer dia da semana    |
