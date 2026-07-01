FROM golang:1.26.4 AS builder

WORKDIR /app

COPY . .

RUN go build -o heimdall ./cmd/heimdall

FROM debian:stable-slim

COPY --from=builder /app/heimdall /heimdall

CMD ["/heimdall"]