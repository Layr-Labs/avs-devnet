# Kurtosis package

> [!WARNING]
> Some features won't be available when starting the devnet via Kurtosis CLI.
> This is because the CLI pre-processes some parts of the args-file before invoking Kurtosis.

## How to run

> [!WARNING]
> Since `Layr-Labs/avs-devnet` is a private repository, you'll need to login with `kurtosis github login` to access it.

[After Kurtosis is installed](../README.md#dependencies), you can run [the default config](kurtosis_package/devnet_params.yaml). This spins up a local Ethereum devnet with a single node and all EigenLayer core contracts deployed. It also includes the [blockscout](https://github.com/blockscout/blockscout) explorer.

```sh
kurtosis run github.com/Layr-Labs/avs-devnet --enclave my_devnet --args-file github.com/kurtosis_package/devnet_params.yaml
```

What follows is a brief tutorial on Kurtosis CLI.
For more information, you can check [the documentation](https://docs.kurtosis.com/).

### Run a custom configuration

To run a different configuration, you can write your own config file and pass it to the package like so:

```sh
kurtosis run github.com/Layr-Labs/avs-devnet --enclave my_devnet --args-file devnet_params.yaml
```

For example configurations, check [`examples`](examples/). For more information on the config file format, check [Configuration](../README.md#configuration).

### Stopping and deleting the devnet

In the past commands, we specified the name of the enclave with the `--enclave` flag.
If no name was specified, Kurtosis will generate a random one.
You can check existing enclaves with:

```sh
kurtosis enclave ls
```

Since we named our enclave `my_devnet`, you can stop it with:

```sh
kurtosis enclave stop my_devnet
```

For destroying a stopped enclave, you can use:

```sh
kurtosis enclave rm my_devnet
```

### Download file artifacts

The devnet can generate various file artifacts (e.g. with contract addresses).
You can see a list by running:

```sh
kurtosis enclave inspect my_devnet
```

To download this data from the Kurtosis engine, use:

```sh
kurtosis files download my_devnet <artifact name>
```

This produces a folder named like the artifact containing its files.
