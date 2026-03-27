BINARY_NAME=gophermart

.PHONY: all
all: build #команда по дефолту

.PHONY: build
build: vet
	go build -o $(BINARY_NAME).exe ./cmd/gophermart

.PHONY: vet
vet: 
	go vet ./...

.PHONY: run
run: build
	$(BINARY_NAME).exe

.PHONY: test
test:
	go test ./...

.PHONY: clean
clean:
	go clean
	del /f /q $(BINARY_NAME).exe