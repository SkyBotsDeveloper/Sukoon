# Testing

## Unit And Service Tests

```powershell
cd bot-core
$env:GOCACHE="$PWD\.gocache"
$env:GOMODCACHE="$PWD\.gomodcache"
go test ./...
```

## Integration Tests

Set:

```powershell
$env:TEST_DATABASE_URL="postgres://sukoon:sukoon@127.0.0.1:5432/sukoon_test?sslmode=disable"
$env:TEST_REDIS_ADDR="127.0.0.1:6379"
$env:TEST_REDIS_DB="0"
```

Then run:

```powershell
go test -tags=integration ./...
```

## Formatting

```powershell
gofmt -w ./cmd ./internal ./migrations
```

## Build Validation

```powershell
go build ./cmd/bot-core
go build ./cmd/migrate
```

## CI

GitHub Actions validates:

- formatting drift
- unit and service tests
- integration tests
- builds for `bot-core` and `migrate`
- Docker image smoke build

Workflow:

- [.github/workflows/bot-core-ci.yml](/c:/Users/strad/OneDrive/Documents/shortcuts/Downloads/Sukoon/.github/workflows/bot-core-ci.yml)

## Coverage Focus

The current suite explicitly protects:

- webhook auth and idempotency
- worker retry and dead-letter behavior
- moderation command correctness
- warn mode and warn limit behavior
- antiflood
- captcha correctness
- owner and federation workflows
- clone isolation and lifecycle
- antiabuse and antibio policy behavior
- bulk removal and username resolution acceptance criteria
- Postgres migration contracts
- Valkey shared-state behavior
