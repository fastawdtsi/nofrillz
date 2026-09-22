FROM golang:1.25.6-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ai-poster ./cmd/ai-poster

FROM scratch

WORKDIR /app

COPY --from=builder /out/ai-poster /app/ai-poster
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY config.yaml /app/config.yaml

ENTRYPOINT ["/app/ai-poster"]
CMD ["/app/config.yaml"]
