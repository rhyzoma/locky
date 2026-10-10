# syntax=docker/dockerfile:1

FROM golang:1.23 AS builder

WORKDIR /src

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /out/locky ./cmd/locky

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /out/locky /app/locky

ENV CONFIG_FILE=/etc/locky/config.yaml \
    GRPC_ADDR=:8080

EXPOSE 8080

USER nonroot:nonroot
ENTRYPOINT ["/app/locky"]
