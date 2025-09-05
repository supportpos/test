FROM golang:1.21-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o notificationservice ./cmd/api

FROM alpine:3.18
WORKDIR /app
COPY --from=build /src/notificationservice ./notificationservice
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://localhost:8080/health || exit 1
ENTRYPOINT ["./notificationservice"]
