# --- Build stage ---
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o events main.go

# --- Runtime stage ---
FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/events /usr/local/bin/events
ENV PORT=8082
EXPOSE 8082
ENTRYPOINT ["/usr/local/bin/events"]
