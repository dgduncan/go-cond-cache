.PHONY : build-image dep lint up down

dep:
	go mod tidy && go mod vendor

lint:
	docker run --rm -v .:/app -w /app golangci/golangci-lint:v1.61.0 golangci-lint run -v ./...

build-image:
	docker build -t go-cond-cache:vlocal .
