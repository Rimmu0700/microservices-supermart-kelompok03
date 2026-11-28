# ---------- STAGE 1: builder ----------
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -g 10001 -S appgroup && adduser -S -D -H -u 10001 -G appgroup appuser

WORKDIR /src

# Optimasi cache: manifes dependensi dulu, source code belakangan
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

# Biner statis (CGO_ENABLED=0); -s -w memangkas symbol table dan DWARF
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" -o /out/app .

# ---------- Varian A: Alpine ----------
FROM alpine:3.19 AS runtime-alpine
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /etc/passwd /etc/group /etc/
COPY --from=builder /out/app /app/app
USER 10001:10001
EXPOSE 3000 50051
ENTRYPOINT ["/app/app"]

# ---------- Varian B: Distroless ----------
FROM gcr.io/distroless/static-debian12:nonroot AS runtime-distroless
COPY --from=builder /out/app /app/app
USER nonroot:nonroot
EXPOSE 3000 50051
ENTRYPOINT ["/app/app"]

# ---------- Varian C: Scratch ----------
FROM scratch AS runtime-scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/group /etc/
COPY --from=builder /out/app /app/app
USER 10001:10001
EXPOSE 3000 50051
ENTRYPOINT ["/app/app"]