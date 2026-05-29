# Stage 1 - Builder
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o scc-service .

# Stage 2 - Production
FROM scratch

WORKDIR /app

COPY --from=builder /app/scc-service .

# COPY CA CERTIFICATES
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 3001

CMD ["./scc-service"]