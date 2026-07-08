# Multi-stage build: compile a static binary, then ship it on a minimal runtime.
FROM golang:1.26-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 vicpay
WORKDIR /app
COPY --from=build /out/api /app/api
COPY migrations /app/migrations
USER vicpay
EXPOSE 8080
ENTRYPOINT ["/app/api"]
