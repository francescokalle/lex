# Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o lex ./cmd/lex

# Runtime stage
FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 lex
WORKDIR /app
COPY --from=builder /build/lex /app/lex
COPY --from=builder /build/site /app/site
COPY deploy/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh && chown -R lex:lex /app
USER lex
VOLUME ["/data"]
EXPOSE 8080
ENTRYPOINT ["/app/entrypoint.sh"]
