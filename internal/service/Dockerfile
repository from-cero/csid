FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" -trimpath \
    -o "/app/server" "./cmd/server"

FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=builder /app/server /server
COPY --from=builder /app/config.yml /config.yml

EXPOSE 8080 9090

ENTRYPOINT ["/server"]
