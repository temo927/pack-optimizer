# Pack Optimizer

Full-stack implementation of the pack calculator challenge.

## Stack
- Backend: Go 1.22, chi, pgx, Redis, hexagonal style
- DB: Postgres
- Cache: Redis
- Frontend: React + Vite

## Quick start (Docker)
```bash
docker compose up --build
```
Services:
- API: http://localhost:8080/api/v1
- Frontend: http://localhost:5173

Run migrations (db is auto-seeded by migration 0001).

## API
- GET `/api/v1/packs` → `{ "sizes": [250,500,1000,2000,5000] }`
- PUT `/api/v1/packs` body `{ "sizes":[23,31,53] }`
- POST `/api/v1/calculate` body `{ "amount": 500000 }` or `{ "amount":500000, "sizes":[23,31,53] }`

## Tests
```bash
cd backend
go test ./...                   # unit tests
go test -tags=integration ./... # integration (requires Docker)
```

## Environment
```
HTTP_PORT=8080
DATABASE_URL=postgres://postgres:postgres@localhost:5432/packs?sslmode=disable
REDIS_ADDR=localhost:6379
```


