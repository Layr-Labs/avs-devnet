.PHONY: deps install start_devnet stop_devnet generate_envscript

deps:
	@echo "Installing dependencies..."
	go mod tidy

install: generate_envscript
	@echo "Installing package..."
	go install ./...
	@-asdf reshim 2> /dev/null
	@echo "Package installed successfully!"
	@echo
	@echo "Remember to run 'source env.sh' to set the environment variables"

KURTOSIS_PACKAGE_PATH:=$(shell cd ../kurtosis_package && pwd -P)

generate_envscript:
	echo "export AVS_DEVNET__PACKAGE_NAME=$(KURTOSIS_PACKAGE_PATH)" > env.sh
	chmod u+x env.sh

devnet.yaml:
	go run cmd/devnet/main.go init

start_devnet: devnet.yaml
	go run cmd/devnet/main.go start

stop_devnet: devnet.yaml
	go run cmd/devnet/main.go stop
