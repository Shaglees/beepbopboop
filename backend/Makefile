.PHONY: run test migrate beepbopgraph install-beepbopgraph

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
