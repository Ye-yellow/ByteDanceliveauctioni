.PHONY: api run test tidy docker-up docker-down docker-logs

GO ?= go
PROTOC ?= /home/ye/OpenClaw/state/tools/protoc/bin/protoc
PROTOC_INC ?= third_party
PROTOC_PLUGIN_PATH ?= /home/ye/go/bin
COMPOSE ?= docker compose

api:
	PATH="$(PROTOC_PLUGIN_PATH):$$PATH" $(PROTOC) \
		--proto_path=. \
		--proto_path=$(PROTOC_INC) \
		--go_out=paths=source_relative:. \
		--go-grpc_out=paths=source_relative:. \
		--go-http_out=paths=source_relative:. \
		api/auction/service/v1/auction.proto \
		api/auction/service/v1/user.proto \
		api/auction/service/v1/shop.proto

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
