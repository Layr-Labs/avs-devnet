.PHONY: help deps install fmt lint test \
	cli_deps generate_envscript cli_start cli_stop cli_fmt cli_lint \
	kurtosis_start kurtosis_stop kurtosis_fmt \
	kurtosis_incredible_squaring kurtosis_hello_world build_hello_world_image

##### Variables #####

KURTOSIS_DIR:=$(shell cd kurtosis_package/ && pwd -P)

CURRENT_COMMIT:=$(shell git describe --always --abbrev=8 --dirty)
INSTALLATION_DIR:=$(shell dirname $$(go list -f '{{.Target}}' cmd/devnet/main.go))
# These flags set some global constants in the build
GO_LDFLAGS:='-X main.version=dev-$(CURRENT_COMMIT) -X github.com/Layr-Labs/avs-devnet/src/cmds/flags.DefaultKurtosisPackage=$(KURTOSIS_DIR)'


##### General #####

help: ## 📚 Show help for each of the Makefile recipes
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

deps: kurtosis_deps cli_deps ## 📥 Install dependencies

install: ## 📦 Install the CLI
	@echo "Installing package..."
	CGO_ENABLED=0 go install -ldflags $(GO_LDFLAGS) -v ./...
	@asdf reshim 2> /dev/null || true
	@echo
	@echo "Installation successfull!"
	@echo "Package was installed to $(INSTALLATION_DIR)"
	@echo $(PATH) | grep -q $(INSTALLATION_DIR) || echo "\nWARNING: $(INSTALLATION_DIR) doesn't seem to be in your PATH.\
		\nIf the command can't be found, try adding to your ~/.bashrc the following line and restarting your shell:\
		\nexport PATH=\"\$$PATH:$(INSTALLATION_DIR)\""

fmt: kurtosis_fmt cli_fmt ## 🧹 Format all code

lint: kurtosis_lint cli_lint ## 🧹 Lint all code

test: ## 🧪 Run tests
	go test -v -timeout 30m ./...


##### CLI #####

cli_deps:
	@echo "Installing Go dependencies..."
	go mod tidy

devnet.yaml:
	$(PACKAGE_ENV_VAR) go run cmd/devnet/main.go init

cli_start: devnet.yaml ## 🚀 Start the devnet (CLI)
	$(PACKAGE_ENV_VAR) go run cmd/devnet/main.go start

cli_stop: devnet.yaml ## 🛑 Stop the devnet (CLI)
	$(PACKAGE_ENV_VAR) go run cmd/devnet/main.go stop

cli_fmt:
	go fmt ./...

cli_lint:
	golangci-lint run


##### Kurtosis Package #####

kurtosis_deps:
	@echo "Checking Kurtosis is installed..."
	@command -v kurtosis 2>&1 > /dev/null || (echo "Kurtosis CLI not found. Please install it from https://docs.kurtosis.com/install/" && exit 1)
	@command -v docker 2>&1 > /dev/null || (echo "Docker not found" && exit 1)

kurtosis_start: ## 🚀 Start the devnet (Kurtosis)
	kurtosis run $(KURTOSIS_DIR) --enclave=devnet --args-file=src/config/default_config.yaml

kurtosis_stop: ## 🛑 Stop the devnet (Kurtosis)
	-kurtosis enclave rm -f devnet

kurtosis_fmt:
	cd $(KURTOSIS_DIR) && kurtosis lint --format

kurtosis_lint:
	cd $(KURTOSIS_DIR) && kurtosis lint


##### Examples #####

# incredible-squaring-avs example

kurtosis_incredible_squaring: ## 🚀 Start the devnet with the Incredible Squaring AVS example (Kurtosis)
	@echo "Starting devnet with incredible_squaring example..."
	kurtosis run $(KURTOSIS_DIR) --enclave=devnet --args-file=examples/incredible_squaring.yaml

# TODO: remove this once we have better testing
check_devnet:
	sleep 60
	# Check that all services are marked as "RUNNING", otherwise fail and print the status
	kurtosis enclave inspect devnet | tee output.txt
	grep -q "STOPPED" output.txt && exit 1 || true

# hello-world-avs example

# Version from 2025-01-04
HELLO_WORLD_REF:=4626e206fd119b26ebe98935b256daa7256e863b

examples/hello-world-avs:
	@echo "Cloning hello-world-avs repo..."
	@mkdir -p examples/hello-world-avs
	@cd examples/hello-world-avs && \
		git init . && \
		git remote add origin https://github.com/Layr-Labs/hello-world-avs.git && \
		git fetch --depth 1 origin $(HELLO_WORLD_REF) && \
		git checkout FETCH_HEAD && \
		git submodule update --init --recursive --depth 1

build_hello_world_image: examples/hello-world-avs
	@echo "Building hello_world docker image..."
	docker build -t hello_world -f examples/hello-world-avs/Dockerfile examples/hello-world-avs

kurtosis_hello_world: build_hello_world_image ## 🚀 Start the devnet with the Hello World AVS example (Kurtosis)
	@echo "Starting devnet with hello_world example..."
	kurtosis run $(KURTOSIS_DIR) --enclave=devnet --args-file=examples/hello_world.yaml
