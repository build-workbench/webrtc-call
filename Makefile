build:
	go build ./...

run:
	go run ./cmd/server

test:
	go test -race -count=1 ./...

vet:
	go vet ./...

fmt:
	go fmt ./...

clean:
	rm -f coverage.out coverage.html

.PHONY: build run test vet fmt clean
