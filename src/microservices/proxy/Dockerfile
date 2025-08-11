FROM golang:1.21 as builder
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o proxy

FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=builder /app/proxy /proxy
EXPOSE 8000
USER nonroot:nonroot
ENTRYPOINT ["/proxy"]

