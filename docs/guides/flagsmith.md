# Feature Flags — Flagsmith

## Como funciona

```
Flagsmith UI → API REST → featureflags.IsEnabled() → rotina habilitada / desabilitada
```

- **Flagsmith**: plataforma de feature flags self-hosted. Armazena flags e as serve via API REST.
- **featureflags.Init()**: inicializa o cliente SDK ao subir o worker, lendo `FLAGSMITH_API_KEY` e `FLAGSMITH_HOST`.
- **featureflags.IsEnabled(flagName)**: consulta a API a cada chamada e retorna `true`/`false`. Retorna `false` em caso de falha (fail-safe).

O padrão demonstrado aqui separa **deploy** (colocar o código em produção) de **release** (ativar a funcionalidade), permitindo ligar/desligar rotinas sem redeploy.

## Setup inicial

> Faça isso uma única vez após subir o Flagsmith pela primeira vez.

1. Subir o Flagsmith:
```sh
sudo docker compose up flagsmith -d
```

2. Acessar [http://localhost:8000](http://localhost:8000) e criar a conta admin.

3. Criar uma organização (ex: `Kaizen`) e um projeto chamado `kaizen-secretary`.

4. Dentro do projeto, entrar no environment **Development** → copiar a **Environment API Key** e colocá-la no `.env`:
```
FLAGSMITH_API_KEY=<sua-key-aqui>
```

5. Criar as flags necessárias (menu **Features**):

| Flag | Tipo | Default |
|---|---|---|
| `scout_monthly_reminder_enabled` | Feature (boolean) | habilitada |
| `dry_run_mode` | Feature (boolean) | desabilitada |

## Passo a passo para demonstrar

1. Subir o projeto completo:
```sh
sudo docker compose up -d
go run cmd/worker/main.go
```

2. Verificar nos logs do worker a linha:
```
"flagsmith initialized"
```

3. **Teste disable** — desabilitar a rotina sem redeploy:
   - No Flagsmith UI, desabilitar a flag `scout_monthly_reminder_enabled`
   - Aguardar o próximo tick do cron (a cada 30s)
   - Verificar nos logs:
   ```
   "routine disabled by feature flag"   routine=PublishScoutMonthlyFees
   ```
   - Reabilitar a flag para restaurar o comportamento normal

4. **Teste dry-run** — executar a lógica sem enviar mensagens reais:
   - No Flagsmith UI, habilitar a flag `dry_run_mode`
   - Aguardar o próximo tick do cron
   - Verificar nos logs (uma linha por destinatário):
   ```
   "[DRY RUN] would publish message"   recipient_phone=... recipient_name=... message_preview=...
   ```
   - Confirmar que nenhum evento foi publicado no Kafka (verificar no [Kafka UI](http://localhost:8080))
   - Desabilitar `dry_run_mode` para retornar ao envio real
