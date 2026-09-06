# fiapx-auth-service

Serviço de autenticação (cadastro/login por usuário e senha) do sistema de
processamento de vídeos FIAP X — Hackathon SOAT (Fase 5). Um dos 4
microsserviços do projeto — ver a
[documentação da arquitetura completa](https://github.com/noggrj/hacktown-fase-5-infra/blob/main/docs/ARQUITETURA.md)
(diagrama, fluxo de eventos, decisões) e os contratos de evento em
[`fiapx-events`](https://github.com/noggrj/hacktown-fase-5-events).

Emite os JWTs (HS256) que os demais serviços (`fiapx-video-service`
principalmente) validam para proteger suas rotas — todos compartilham o
mesmo `JWT_SECRET` via Kubernetes Secret.

## Arquitetura interna

Clean Architecture, mesma convenção usada nos demais serviços do projeto:

```
cmd/api/main.go                    → wiring, graceful shutdown
internal/auth/domain/              → entidade User, regras de validação, erros de domínio
internal/auth/domain/repository.go → interface UserRepository (porta)
internal/auth/gateway/             → implementação Postgres da porta acima
internal/auth/usecase/             → RegisterUseCase, LoginUseCase
internal/auth/delivery/http/       → handlers Chi (POST /auth/register, /login, GET /auth/me)
internal/platform/                 → config, logging, db, health, metrics, jwt, httpauth (transversal)
```

`internal/platform/jwt` e `internal/platform/httpauth` existem aqui e serão
duplicados (não importados como dependência) nos outros serviços que
precisam validar o token — são ~80 linhas ao todo, pequeno demais para
justificar um módulo Go compartilhado à parte.

## Endpoints

| Método | Rota | Auth | Descrição |
|---|---|---|---|
| POST | `/auth/register` | — | Cria usuário (`email`, `password`, mín. 8 caracteres) |
| POST | `/auth/login` | — | Retorna `{"token": "..."}` (JWT válido por 24h) |
| GET | `/auth/me` | Bearer | Retorna `userId`/`email` extraídos do token — usado para validar o fluxo no vídeo de apresentação |
| GET | `/health` | — | Liveness |
| GET | `/ready` | — | Readiness (checa Postgres) |
| GET | `/metrics` | — | Scrape Prometheus |

## Rodando localmente

```bash
cp .env.example .env
# edite DB_URL e JWT_SECRET

docker run --name fiapx-auth-db -e POSTGRES_USER=auth -e POSTGRES_PASSWORD=auth \
  -e POSTGRES_DB=fiapx_auth -p 5433:5432 -d postgres:16-alpine
psql "postgres://auth:auth@localhost:5433/fiapx_auth?sslmode=disable" -f migrations/0001_create_users.sql

go run ./cmd/api
```

## Testes

```bash
go test ./...                              # suíte completa
go test -coverprofile=coverage.out \
  ./internal/auth/domain/... ./internal/auth/usecase/... ./internal/auth/delivery/... \
  ./internal/platform/jwt/... ./internal/platform/httpauth/... ./internal/platform/health/... \
  ./internal/platform/config/... ./internal/platform/metrics/...
go tool cover -func=coverage.out | tail -1
```

O gate de cobertura no CI (`.github/workflows/ci.yml`, mínimo 70%) é medido
só sobre os pacotes com lógica de negócio real — `cmd/api` (wiring),
`internal/platform/db` e `internal/auth/gateway` (precisam de um Postgres
de verdade para um teste significativo) e `internal/platform/logging`
(wrapper de 3 linhas) ficam de fora do gate por não serem testáveis de
forma unitária com proveito.

## Deploy

Imagem via `Dockerfile` (multi-stage). Manifests em `k8s/base/` (namespace,
configmap, deployment, service) aplicados com `kubectl apply -f k8s/base/`.

`k8s/secret.example.yaml` fica **fora** de `k8s/base/` de propósito — é só
um exemplo do shape esperado do Secret `fiapx-auth-secret`; se morasse
dentro de `k8s/base/`, um `kubectl apply -f k8s/base/` sobrescreveria o
secret real com os placeholders a cada deploy (erro já cometido e corrigido
em `autorepair-billing-service` na Fase 4).
