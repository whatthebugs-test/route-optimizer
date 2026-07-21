# route-optimizer

Builds pickup-and-delivery routes for each terminal's day. dispatch-service
calls it when the morning board is published and whenever a dispatcher hits
"re-plan". Routes are sequenced with nearest-neighbour construction plus a
2-opt pass, split by truck capacity and a stop limit.

## Layout

```
cmd/optimizer/        HTTP server (gin)
internal/solver/      route construction and improvement (pure, unit tested)
internal/store/       Postgres access (pgx): stops in, route plans out
internal/cache/       Redis memoisation of solved plans
internal/api/         handlers and service-token auth
migrations/           tables owned by this service in the dispatch database
```

## Running locally

```sh
export DATABASE_URL=postgres://dispatch:dispatch@localhost:5432/dispatch
export JWT_SECRET=dev-only-secret
export LOG_PRETTY=1
go run ./cmd/optimizer
```

```sh
curl -s localhost:8090/v1/routes/optimize \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"terminals":["482"],"day":"2026-11-18","capacity_lbs":10000,"max_stops":18}'
```

`$TOKEN` is a shipment-api service token with the `routes:plan` scope.

## Tests

```sh
go test ./...
```

## Environment

| Variable | Default | Notes |
| --- | --- | --- |
| `DATABASE_URL` | | dispatch Postgres |
| `REDIS_URL` | `redis://localhost:6379/1` | plan cache |
| `JWT_SECRET` | | shared with shipment-api |
| `PORT` | `8090` | |

## Deploy

ECS service `route-optimizer` (cluster `prod-apps`), internal ALB only.
