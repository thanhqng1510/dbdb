FROM golang:1.24-alpine AS builder

RUN apk add --no-cache make

WORKDIR /app
COPY . .

RUN make build

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/bin/dbdb /app/dbdb

VOLUME ["/app/data"]
EXPOSE 8000-9000

ENTRYPOINT ["/app/dbdb"]