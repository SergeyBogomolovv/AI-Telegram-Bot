FROM golang:1.24.5-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o aibot *.go

FROM gcr.io/distroless/base-debian12

WORKDIR /app

COPY --from=builder /app/aibot ./aibot

USER nonroot:nonroot

ENTRYPOINT ["./aibot"]
