.PHONY: start_devnet stop_devnet clean_devnet

start_devnet:
	kurtosis run kurtosis_package/ --enclave=devnet

stop_devnet:
	kurtosis enclave stop devnet

clean_devnet: stop_devnet
	kurtosis enclave rm devnet
