FROM golang:1.22 as builder
LABEL maintainer="Nikolai Akulov"

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o app ./cmd/main.go

FROM alpine:latest

RUN adduser -D -u 10001 appuser

WORKDIR /home/appuser

COPY --from=builder /app/app ./app
RUN chown appuser:appuser ./app && chmod +x ./app

USER appuser

EXPOSE 8080

CMD ["./app"]