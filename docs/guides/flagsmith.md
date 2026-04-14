# Feature Flags — Flagsmith

## Como funciona

```
app → Flagsmith SDK (local evaluation) → Flagsmith API → Dashboard
```

- **Flagsmith**: plataforma open-source de feature flags e remote config. A instância roda localmente via Docker, com banco Postgres próprio.
- **Local evaluation**: o SDK sincroniza o estado das flags em background (a cada 5 s) e avalia localmente — sem latência de rede por flag consultada.
- **Environment API Key**: chave que identifica o ambiente (ex: *Development*, *Production*) dentro de um projeto Flagsmith. É ela que o SDK usa para se autenticar e buscar as flags.

## Setup inicial

### 1. Subir os serviços

```sh
sudo docker compose up -d flagsmith-database flagsmith
```

O painel estará disponível em [http://localhost:8000](http://localhost:8000).

### 2. Criar conta e projeto

1. Acesse [http://localhost:8000](http://localhost:8000) e registre uma conta (o registro sem convite está habilitado).
2. Crie uma **Organisation** e, dentro dela, um **Project** (ex: `kaizen-secretary`).
3. O Flagsmith já cria dois ambientes por padrão: **Development** e **Production**.

### 3. Obter a API Key do ambiente

1. No projeto, vá em **SDK Keys → Server-side environment keys → Crie uma chave**.
2. Copie o valor de **Environment API Key** — ela tem o formato `ser.<hash>`.
3. Passe essa chave ao inicializar o client na aplicação:

```go
fs := flagsmith.NewClient("ser.<sua-chave>")
```

### 4. Criar uma feature flag

1. No painel, acesse **Features → Create Feature**.
2. Defina um **Feature Name** (ex: `routine_mensalidade_escoteiro`) — use snake_case.
3. Opcionalmente, defina um **Value** (string/number/boolean) para usar como remote config.
4. Salve e ative a flag no ambiente desejado.

## Usando na aplicação

O client (`internal/flagsmith`) expõe três métodos:

### Verificar se uma flag está habilitada

```go
if fs.IsEnable("routine_mensalidade_escoteiro") {
    // executa a rotina
}
```

### Ler um valor de remote config

```go
delay := fs.GetString("mensalidade_delay_seconds", "30")
```

O segundo argumento é o valor padrão caso a flag não exista ou ocorra erro.

### Inspecionar todas as flags (debug)

```go
fs.PrintAllFlags()
```

## Exemplo no worker

```go
// cmd/worker/main.go
fs := flagsmith.NewClient("ser.5zKKcmwRMHb6jYqMCPjEYW")

c.AddFunc("*/30 * * * * *", func() {
    if !fs.IsEnable("routine_mensalidade_escoteiro") {
        zap.L().Info("routine_mensalidade_escoteiro desabilitada, pulando")
        return
    }
    // ...
})
```

Alterar o estado da flag no painel reflete na aplicação em até **5 segundos**, sem restart.

## Passo a passo para demonstrar

1. Subir o projeto
```sh
sudo docker compose up -d
go run cmd/worker/main.go
```

2. Acessar o [painel do Flagsmith](http://localhost:8000) e ativar/desativar a flag `routine_mensalidade_escoteiro`.

3. Observar nos logs que a rotina é executada ou ignorada conforme o estado da flag:
```
{container="kaizen-secretary"} | json | logger="flagsmith"
```
