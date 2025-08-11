# --- build ---
FROM golang:1.21-alpine AS builder
WORKDIR /app
# если у тебя есть go.sum — скопируй и его, это ускорит кэш
COPY go.mod ./
# COPY go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o /out/proxy ./main.go

# --- runtime ---
FROM gcr.io/distroless/static-debian12
COPY --from=builder /out/proxy /proxy
EXPOSE 8000
USER nonroot:nonroot
ENTRYPOINT ["/proxy"]




