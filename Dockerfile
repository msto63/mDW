FROM golang:1.21-alpine AS builder

ARG SERVICE=api-gateway

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /service ./cmd/${SERVICE}

FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /service .

EXPOSE 8080

ENTRYPOINT ["./service"]
