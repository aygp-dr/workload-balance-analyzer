.PHONY: build run test clean

build:
	go build -o bin/workload-balance-analyzer .

run: build
	./bin/workload-balance-analyzer

test:
	go test ./...

clean:
	rm -rf bin/
