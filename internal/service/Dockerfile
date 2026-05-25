FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" -trimpath \
    -o /app/abc ./cmd/abc

FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=builder /app/abc /abc
COPY --from=builder /app/config.yml /config.yml

EXPOSE 8080 9090

ENTRYPOINT ["/abc"]
