# Satu Dockerfile untuk SEMUA service. Pilih service lewat build arg SERVICE
# (nama folder di backend/cmd), contoh: --build-arg SERVICE=order-service
FROM golang:1.26-alpine AS builder

ARG SERVICE
WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN test -n "$SERVICE" || (echo "build arg SERVICE wajib diisi" && exit 1)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/service ./cmd/${SERVICE}

# Runtime stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /out/service /app/service

EXPOSE 3000

CMD ["/app/service"]