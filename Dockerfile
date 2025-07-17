FROM golang:1.22 as builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o app ./cmd/main.go

FROM alpine:latest

RUN adduser -D -u 10001 appuser

WORKDIR /home/appuser

COPY --from=builder /app/app ./app
RUN chown appuser:appuser ./app && chmod +x ./app

USER appuser

EXPOSE 8080

CMD ["./app"]