FROM golang:1.22-alpine AS builder

ENV PATH="/usr/local/go/bin:${PATH}"

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/auction-service ./app/auction/service/cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/auction-shard-gateway ./app/auction/service/cmd/shard_gateway

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /out/auction-service /app/auction-service
COPY --from=builder /out/auction-shard-gateway /app/auction-shard-gateway
COPY app/auction/service/configs/config.yaml /app/configs/config.yaml

ENV AUCTION_HTTP_ADDR=:18080
EXPOSE 18080 18081

ENTRYPOINT ["/app/auction-service"]
