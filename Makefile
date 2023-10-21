BINDIR := $(CURDIR)/bin

test:
	go mod tidy
	go test ./... -shuffle=on -race

lint:
	go mod tidy
	go vet  ./...

test.cover:
	go mod tidy
	go test -race -shuffle=on -coverprofile=coverage.txt -covermode=atomic ./...
