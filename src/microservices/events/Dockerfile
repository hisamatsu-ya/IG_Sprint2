# ---- build ----
FROM golang:1.21-alpine AS build
WORKDIR /src

# сначала только go.mod (можно без go.sum)
COPY go.mod ./
RUN go mod download

# теперь копируем код — чтобы tidy увидел импорты
COPY . .

# сгенерируем/обновим go.sum и подтянем всё нужное
RUN go mod tidy

# собираем бинарь
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/events ./main.go

# ---- runtime ----
FROM gcr.io/distroless/base-debian12
WORKDIR /
USER nonroot:nonroot
COPY --from=build /out/events /events
ENV PORT=8082
EXPOSE 8082
ENTRYPOINT ["/events"]

