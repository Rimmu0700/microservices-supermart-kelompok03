# promotion-service - Dockerfile MILIK service ini (tidak dipakai bersama service lain)
# Multi-stage: builder (SDK Go) -> runtime minimal non-root.
FROM golang:1.26-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/service ./cmd/promotion-service

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata \
 && addgroup -g 10001 -S appgroup && adduser -S -D -H -u 10001 -G appgroup appuser
COPY --from=builder /out/service /app/service
USER 10001:10001
EXPOSE 3000
HEALTHCHECK --interval=10s --timeout=5s --start-period=20s --retries=6 \
  CMD wget -qO- http://127.0.0.1:3000/api/v1/health >/dev/null || exit 1
ENTRYPOINT ["/app/service"]
