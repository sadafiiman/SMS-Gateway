.PHONY: run build test test-race lint fmt vet docker-build docker-up docker-down

run:
	go run ./cmd/api

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/sms-gateway ./cmd/api

test:
	go test ./... -cover

test-race:
	go test ./... -race -cover

fmt:
	gofmt -l .

vet:
	go vet ./...

lint: fmt vet

docker-build:
	docker build -t sms-gateway:local .

docker-up:
	docker compose up --build

docker-down:
	docker compose down
