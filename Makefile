.PHONY: start_devnet stop_devnet clean_devnet format \
	start_helloworld

start_devnet:
	kurtosis run kurtosis_package/ --enclave=devnet --args-file=kurtosis_package/devnet_params.yaml

stop_devnet:
	-kurtosis enclave stop devnet

clean_devnet: stop_devnet
	-kurtosis enclave rm devnet

format:
	kurtosis lint --format

# hello-world-avs example

examples/hello-world-avs:
	@echo "Cloning hello-world-avs repo..."
	@mkdir -p examples/hello-world-avs
	@cd examples/hello-world-avs && \
		git init . && \
		git remote add origin https://github.com/Layr-Labs/hello-world-avs.git && \
		git fetch --depth 1 origin bbd6ea74d7ee419c77b3a1983ff5f1685aac5231 && \
		git checkout FETCH_HEAD && \
		git submodule update --init --recursive --depth 1

build_helloworld_image: examples/hello-world-avs
	@echo "Building helloworld docker image..."
	docker build -t helloworld -f examples/hello-world-avs/Dockerfile examples/hello-world-avs

start_helloworld: build_helloworld_image
	@echo "Starting devnet with helloworld example..."
	kurtosis run kurtosis_package/ --enclave=devnet --args-file=examples/hello_world.yaml
