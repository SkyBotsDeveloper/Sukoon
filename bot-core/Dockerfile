FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/bot-core ./cmd/bot-core
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/migrate ./cmd/migrate

FROM alpine:3.22

RUN addgroup -S sukoon && adduser -S -G sukoon sukoon
WORKDIR /app

COPY --from=builder /out/bot-core /usr/local/bin/bot-core
COPY --from=builder /out/migrate /usr/local/bin/migrate
USER sukoon

EXPOSE 8080

CMD ["bot-core"]
