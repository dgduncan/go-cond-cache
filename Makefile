.PHONY : build-image dep lint test integration integration-ci up down

dep:
	go mod tidy && go mod vendor

lint:
	docker run --rm -v .:/app -w /app golangci/golangci-lint:v2.1.6 golangci-lint run -v ./...

test:
	go test -v ./...

build-image:
	docker build -t go-cond-cache:vlocal .

integration: down up
	go test --tags=integration -coverprofile=integration_coverage.out -v ./...
	make down

integration-ci: up
	go test --tags=integration -coverprofile=integration_coverage.out -v ./...

up:
	docker compose up -d

down:
	docker compose down -v
