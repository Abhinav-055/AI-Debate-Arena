FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/worker ./cmd/worker

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates wget procps

COPY --from=builder /out/server /app/server
COPY --from=builder /out/worker /app/worker

EXPOSE 8080
