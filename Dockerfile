FROM golang:1.27 AS builder
ARG CGO_ENABLED=0
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build

FROM alpine AS deployment
COPY --from=builder /app/pricemonitor /pricemonitor
ENTRYPOINT ["/pricemonitor"]
