APP=bot-core

.PHONY: run test fmt tidy build migrate test-integration verify

run:
	go run ./cmd/$(APP)

test:
	go test ./...

fmt:
	gofmt -w ./cmd ./internal ./migrations

tidy:
	go mod tidy

build:
	go build ./cmd/$(APP) && go build ./cmd/migrate

migrate:
	go run ./cmd/migrate

test-integration:
	go test -tags=integration ./...

verify:
	gofmt -w ./cmd ./internal ./migrations
	go test ./...
	go build ./cmd/$(APP) && go build ./cmd/migrate
