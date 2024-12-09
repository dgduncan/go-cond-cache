.PHONY : build-image dep lint integration up down

dep:
	go mod tidy && go mod vendor

lint:
	docker run --rm -v .:/app -w /app golangci/golangci-lint:v1.61.0 golangci-lint run -v ./...

build-image:
	docker build -t go-cond-cache:vlocal .

integration: down up
	go test --tags=integration -v ./...
	make down

up:
	docker compose up -d

down:
	docker compose down -v
