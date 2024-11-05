.PHONY: start_devnet stop_devnet clean_devnet format \
	start_incredible_squaring start_hello_world build_hello_world_image

start_devnet:
	kurtosis run kurtosis_package/ --enclave=devnet --args-file=kurtosis_package/devnet_params.yaml

stop_devnet:
	-kurtosis enclave stop devnet

clean_devnet: stop_devnet
	-kurtosis enclave rm devnet

format:
	kurtosis lint --format

# incredible-squaring-avs example

start_incredible_squaring:
	@echo "Starting devnet with incredible_squaring example..."
	kurtosis run kurtosis_package/ --enclave=devnet --args-file=examples/incredible_squaring.yaml

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

start_hello_world: examples/hello-world-avs
	@echo "Starting devnet with hello_world example..."
	kurtosis run kurtosis_package/ --enclave=devnet --args-file=examples/hello_world.yaml
