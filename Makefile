.PHONY: build run test lint clean test-containers clean-test-containers

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

test-containers:
	@echo "Creating test containers with known exit codes..."
	@docker rm -f kernus-test-clean kernus-test-crash kernus-test-killed 2>/dev/null; true
	@docker run --name kernus-test-clean alpine sh -c "exit 0"
	@docker run --name kernus-test-crash alpine sh -c "exit 1"
	@docker run --name kernus-test-killed alpine sh -c "exit 137"
	@echo "✓ Created: kernus-test-clean (exit 0 / clean_stop)"
	@echo "✓ Created: kernus-test-crash (exit 1 / app_crashed)"
	@echo "✓ Created: kernus-test-killed (exit 137 / force_killed)"
	@echo "  Run 'kernus agent start' to send metrics with exit info to the backend."

clean-test-containers:
	@docker rm -f kernus-test-clean kernus-test-crash kernus-test-killed 2>/dev/null; true
	@echo "✓ Test containers removed."
