APP_NAME := cartero
DIST_DIR := dist

.PHONY: bootstrap fmt vet test build smoke clean docker-build

bootstrap:
	go mod tidy

fmt:
	gofmt -w ./cmd ./internal

vet:
	go vet ./...

test:
	go test ./...

build:
	mkdir -p $(DIST_DIR)
	go build -o $(DIST_DIR)/$(APP_NAME) ./cmd/cartero

smoke:
	bash ./scripts/smoke.sh

docker-build:
	docker build -t $(APP_NAME):dev .

clean:
	rm -rf $(DIST_DIR)
	
