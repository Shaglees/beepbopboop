.PHONY: run test migrate beepbopgraph install-beepbopgraph docker-build docker-up docker-down docker-logs

run:
	go run ./cmd/server

test:
	go test ./... -v -count=1

migrate:
	go run ./cmd/server -migrate

beepbopgraph:
	go build -o beepbopgraph ./cmd/beepbopgraph

install-beepbopgraph: beepbopgraph
	cp beepbopgraph /usr/local/bin/

docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f server
