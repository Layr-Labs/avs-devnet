.PHONY: start_devnet stop_devnet clean_devnet

start_devnet:
	kurtosis run kurtosis_package/ --enclave=devnet --args-file=kurtosis_package/devnet_params.yaml

stop_devnet:
	-kurtosis enclave stop devnet

clean_devnet: stop_devnet
	-kurtosis enclave rm devnet
