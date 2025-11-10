# Makefile for SafelyYou Fleet project

.PHONY: install run unit-test integration-test tests

# Load variables from .env if it exists
ifneq (,$(wildcard .env))
  include .env
  export
endif


## install: install everything needed to run the project
install:
	@echo "Fetching Swagger dependencies..."
	@go get github.com/swaggo/swag@latest
	@go get github.com/swaggo/files@latest
	@go get github.com/swaggo/gin-swagger@latest
	@echo "Running go mod tidy..."
	@go mod tidy
	@echo "Installing swag CLI..."
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Done."


## run: start the HTTP server on PORT (default 8080)
run:
	go run ./cmd/app

#create the doc
doc:
	@echo "Generating Swagger API documentation..."
	@swag init -g cmd/app/main.go -d . -o docs
	@echo "Documentation generated in ./docs"

#run all the tests
test:
	@echo "Running tests..."
	go test ./... -v

#run the simulate device (
simulate:
	@echo "Running simulate device on port $(PORT)..."
	$(DEVICE_SIM_BIN) --port $(PORT)
