# Rate Limiter (Desafio Técnico - Pós Golang Full Cycle)

Rate Limiter em Go, implementado como middleware HTTP, com limitação por IP e por Token de acesso (Token com precedência sobre IP), usando Redis para persistência via padrão **Strategy**.

## Arquitetura

```
cmd/server/main.go          -> composição da aplicação (wiring)
internal/config             -> carga de configuração via env/.env
internal/limiter
  strategy.go                -> interface Strategy (contrato de persistência)
  redis_strategy.go           -> implementação Strategy usando Redis
  limiter.go                  -> regras de negócio (Limiter), independente de HTTP e de Redis
internal/middleware
  ratelimiter.go              -> middleware HTTP, desacoplado da lógica de negócio
test/bdd
  features/rate_limiter.feature -> cenários Gherkin (BDD) end-to-end
  rate_limiter_test.go           -> runner godog (TestFeatures)
  steps_test.go                  -> step definitions
  run-e2e.sh                     -> sobe containers, roda os testes e2e e os encerra
```

- **Strategy**: `limiter.Strategy` define `Increment`, `IsBlocked` e `Block`. `RedisStrategy` é a implementação obrigatória do desafio. Para trocar de mecanismo de persistência (ex.: memória, Postgres, Memcached), basta criar um novo tipo que implemente `Strategy` e injetá-lo em `limiter.New(strategy, cfg)` — nenhuma outra parte do código muda.
- **Desacoplamento**: `Limiter` (regra de negócio) não conhece HTTP; `middleware.RateLimiter` (transporte) não conhece Redis, apenas chama `Limiter.Allow(ctx, ip, token)`.
- **Precedência Token > IP**: em `Limiter.resolveRule`, se um token for informado, suas regras (específicas ou padrão) sempre substituem as regras de IP.

## Configuração (variáveis de ambiente / `.env`)

| Variável | Descrição | Padrão |
|---|---|---|
| `SERVER_PORT` | Porta do servidor HTTP | `8080` |
| `REDIS_HOST` | Host do Redis | `localhost` |
| `REDIS_PORT` | Porta do Redis | `6379` |
| `REDIS_PASSWORD` | Senha do Redis | (vazio) |
| `REDIS_DB` | Índice do banco Redis | `0` |
| `RATE_LIMIT_IP_MAX_REQUESTS` | Máximo de requisições/segundo por IP | `10` |
| `RATE_LIMIT_IP_BLOCK_DURATION_SECONDS` | Tempo de bloqueio (s) ao exceder limite por IP | `300` |
| `RATE_LIMIT_TOKEN_MAX_REQUESTS` | Limite padrão (req/s) para tokens sem regra específica | `100` |
| `RATE_LIMIT_TOKEN_BLOCK_DURATION_SECONDS` | Bloqueio padrão (s) para tokens sem regra específica | `300` |
| `RATE_LIMIT_TOKEN_RULES` | Regras específicas por token: `token:limite:bloqueioSegundos;token2:limite:bloqueioSegundos` | (vazio) |

Copie `.env.example` para `.env` e ajuste os valores. Um `.env` de exemplo funcional já está incluído na raiz.

O token é enviado no header: `API_KEY: <TOKEN>`.

## Como executar

### Com Docker (recomendado)

```powershell
docker compose up --build
```

Sobe o Redis e a aplicação na porta `8080`. Teste com:

```powershell
curl.exe -i http://localhost:8080/
curl.exe -i -H "API_KEY: abc123" http://localhost:8080/
```

Ao exceder o limite, a resposta é `429` com o corpo:
`you have reached the maximum number of requests or actions allowed within a certain time frame`

### Localmente (sem Docker)

1. Suba um Redis (`docker run -p 6379:6379 redis:7-alpine`).
2. Ajuste `.env` com `REDIS_HOST=localhost`.
3. Rode:

```powershell
go run ./cmd/server
```

## Como trocar a estratégia de persistência

1. Crie um tipo que implemente `limiter.Strategy`:

```go
type MinhaStrategy struct{ /* ... */ }

func (s *MinhaStrategy) Increment(ctx context.Context, key string, window time.Duration) (int64, error) { /* ... */ }
func (s *MinhaStrategy) IsBlocked(ctx context.Context, key string) (bool, error) { /* ... */ }
func (s *MinhaStrategy) Block(ctx context.Context, key string, duration time.Duration) error { /* ... */ }
```

2. Em `cmd/server/main.go`, troque `limiter.NewRedisStrategy(redisClient)` pela sua implementação ao chamar `limiter.New(strategy, cfg.Limiter)`.

## Testes automatizados

Os testes usam `miniredis` (Redis em memória) para validar `RedisStrategy` e `Limiter` sem depender de infraestrutura externa:

```powershell
go test ./... -v
```

Cobertura:
- Requisições dentro do limite por IP são permitidas.
- Requisições acima do limite por IP retornam bloqueio.
- Bloqueio persiste durante o tempo configurado e libera após expirar.
- **Precedência Token > IP**: mesmo com IP limitado a 1 req/s, um token com limite maior permite mais requisições.
- Token sem regra específica usa o limite/bloqueio padrão de token.
- IPs diferentes são controlados de forma independente.
- Middleware HTTP retorna `429` e a mensagem exata ao exceder o limite, tanto por IP quanto por token.

### Testes BDD (Gherkin + godog) — end-to-end

Além dos testes acima (rápidos, com `miniredis`), há cenários Gherkin em [test/bdd/features/rate_limiter.feature](test/bdd/features/rate_limiter.feature) que validam a API **real**, rodando contra os containers do `docker compose` (app + Redis reais).

Esses testes ficam pulados (`SKIP`) por padrão — inclusive rodando `go test ./...` ou pelo Test Explorer/CodeLens do VS Code — a menos que a variável de ambiente `E2E=1` esteja definida. Isso evita falhas quando os containers não estão no ar, tanto no terminal quanto no runner de testes da IDE.

#### Opção 1 (recomendada): script `run-e2e.sh`

[test/bdd/run-e2e.sh](test/bdd/run-e2e.sh) automatiza o ciclo completo: sobe os containers, roda os testes e2e, aguarda um `ENTER` para você conferir o resultado no terminal e então encerra os containers automaticamente (mesmo se os testes falharem).

```bash
./test/bdd/run-e2e.sh
```

Requer Git Bash (ou WSL) no Windows. Se necessário, dê permissão de execução uma vez: `chmod +x test/bdd/run-e2e.sh`.

#### Opção 2: passo a passo manual

```powershell
docker compose up --build -d
$env:E2E = "1"
go test ./test/bdd/... -v
docker compose down
```

Cada cenário usa um IP forjado exclusivo (header `X-Forwarded-For`) para não interferir com outros cenários/execuções. Cenários cobertos: requisições dentro do limite por IP, estouro do limite por IP retornando `429` com a mensagem exata, e precedência do token sobre o limite de IP.
