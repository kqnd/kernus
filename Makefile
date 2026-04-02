.PHONY: build run test lint clean

build:
	go build -o kernus ./...

run:
	go run . see

test:
	go test ./... -v -race

lint:
	golangci-lint run ./...

clean:
	rm -f kernus kernus.exe
