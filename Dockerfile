# Build stage
FROM golang:1.27-alpine AS builder

WORKDIR /app

# Copy dependency manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build lightweight static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/gemini-web-proxy ./cmd/proxy

# Final runtime stage
FROM scratch

# Copy SSL certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary from builder
COPY --from=builder /bin/gemini-web-proxy /bin/gemini-web-proxy

EXPOSE 58120

ENTRYPOINT ["/bin/gemini-web-proxy"]
