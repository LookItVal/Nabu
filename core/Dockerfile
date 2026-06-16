FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/core-api ./cmd/api

FROM alpine:3.22

RUN addgroup -S app && adduser -S app -G app && apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /out/core-api /app/core-api

EXPOSE 8080
USER app

CMD ["/app/core-api"]
