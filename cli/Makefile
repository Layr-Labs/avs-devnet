.PHONY: deps install start_devnet stop_devnet

deps:
	@echo "Installing dependencies..."
	go mod tidy

install:
	@echo "Installing package..."
	go install ./...
	@-asdf reshim 2> /dev/null

devnet.yaml:
	go run cmd/devnet/main.go init

start_devnet: devnet.yaml
	go run cmd/devnet/main.go start

stop_devnet: devnet.yaml
	go run cmd/devnet/main.go stop
