FROM golang:1.22-alpine AS builder

ENV PATH="/usr/local/go/bin:${PATH}"

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/auction-service ./app/auction/service/cmd/server

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /out/auction-service /app/auction-service
COPY app/auction/service/configs/config.yaml /app/configs/config.yaml

ENV AUCTION_HTTP_ADDR=:18080
EXPOSE 18080

ENTRYPOINT ["/app/auction-service"]
