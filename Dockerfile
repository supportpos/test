FROM golang:1.21-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o api ./cmd/api && CGO_ENABLED=0 go build -o consumer ./cmd/consumer && CGO_ENABLED=0 go build -o migrate ./cmd/migrate

FROM alpine:3.18
WORKDIR /app
COPY --from=build /src/api ./api
COPY --from=build /src/consumer ./consumer
COPY --from=build /src/migrate ./migrate
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://localhost:8080/healthz || exit 1
ENTRYPOINT ["./api"]
