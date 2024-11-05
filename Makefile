.PHONY: help deps install fmt lint \
	cli_deps generate_envscript cli_start_devnet cli_stop_devnet cli_fmt cli_lint \
	kurtosis_start_devnet kurtosis_stop_devnet kurtosis_fmt \
	kurtosis_incredible_squaring kurtosis_hello_world build_hello_world_image

##### Variables #####

CLI_DIR:=cli/
KURTOSIS_DIR:=kurtosis_package/
KURTOSIS_VERSION:=$(shell kurtosis version 2> /dev/null)


##### General #####

help: ## 📚 Show help for each of the Makefile recipes
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

deps: kurtosis_deps cli_deps ## 📥 Install dependencies

install: generate_envscript ## 📦 Install the CLI
	@echo "Installing package..."
	cd $(CLI_DIR) && go install ./...
	@-asdf reshim 2> /dev/null
	@echo "Package installed successfully!"
	@echo
	@echo "Remember to run 'source env.sh' to set the environment variables"

fmt: kurtosis_fmt cli_fmt ## 🧹 Format all code

lint: kurtosis_lint cli_lint ## 🧹 Lint all code


##### CLI #####

cli_deps:
	@echo "Installing Go dependencies..."
	cd $(CLI_DIR) && go mod tidy

generate_envscript:
	echo "export AVS_DEVNET__PACKAGE_NAME=$(shell cd $(KURTOSIS_DIR) && pwd -P)" > env.sh
	chmod u+x env.sh

devnet.yaml:
	cd $(CLI_DIR) && go run cmd/devnet/main.go init

cli_start_devnet: devnet.yaml ## 🚀 Start the devnet (CLI)
	cd $(CLI_DIR) && go run cmd/devnet/main.go start

cli_stop_devnet: devnet.yaml ## 🛑 Stop the devnet (CLI)
	cd $(CLI_DIR) && go run cmd/devnet/main.go stop

cli_fmt:
	cd $(CLI_DIR) && go fmt ./...

cli_lint:
	cd $(CLI_DIR) && golangci-lint run


##### Kurtosis Package #####

kurtosis_deps:
	@echo "Checking Kurtosis is installed..."
	@command -v kurtosis 2>&1 > /dev/null || (echo "Kurtosis CLI not found. Please install it from https://docs.kurtosis.com/install/" && exit 1)
	@command -v docker 2>&1 > /dev/null || (echo "Docker not found" && exit 1)

kurtosis_start_devnet: ## 🚀 Start the devnet (Kurtosis)
	kurtosis run $(KURTOSIS_DIR) --enclave=devnet --args-file=$(KURTOSIS_DIR)/devnet_params.yaml

kurtosis_stop_devnet: ## 🛑 Stop the devnet (Kurtosis)
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

# hello-world-avs example

HELLO_WORLD_REF:=9b8231b16c8bacd4a5eb67e8faa389cd8b1e9600

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
