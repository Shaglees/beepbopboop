.PHONY: run test migrate

run:
	go run ./cmd/server

test:
	go test ./... -v -count=1

migrate:
	go run ./cmd/server -migrate
