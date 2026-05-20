.PHONY: api run test tidy docker-up docker-down docker-logs

GO ?= go
PROTOC ?= protoc
PROTOC_INC ?= third_party
COMPOSE ?= docker compose

api:
	$(PROTOC) \
		--proto_path=. \
		--proto_path=$(PROTOC_INC) \
		--go_out=paths=source_relative:. \
		--go-grpc_out=paths=source_relative:. \
		--go-http_out=paths=source_relative:. \
		api/auction/service/v1/auction.proto \
		api/auction/service/v1/user.proto

run:
	$(GO) run ./app/auction/service/cmd/server

test:
	$(GO) test ./...

tidy:
	$(GO) mod tidy

docker-up:
	cd deploy && $(COMPOSE) up -d --build

docker-down:
	cd deploy && $(COMPOSE) down

docker-logs:
	cd deploy && $(COMPOSE) logs -f --tail=200
