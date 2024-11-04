# EigenLayer AVS Devnet

*AvsDevnet* is a library and CLI tool to start local devnets with specific operator states.
We expect the library to be commonly used in place of mocks for automated testing of specific situations.
The CLI tool, on the other hand, we expect to be used in place of bash scripts for end-to-end testing and local development.

> [!WARNING]  
> Currently, only the Kurtosis package is available.
> Future versions may include the testing library and a CLI.

## Dependencies

Since the Devnet is implemented as a Kurtosis package, we require Kurtosis to be installed.
For how to install it, you can check [here](https://docs.kurtosis.com/install/).

## How to use

> [!WARNING]  
> Since `Layr-Labs/avs-devnet` is a private repository, you'll need to login with `kurtosis github login` to access it.

[After Kurtosis is installed](#dependencies), you can run [the default config](kurtosis_package/devnet_params.yaml). This spins up a local Ethereum devnet with a single node and all EigenLayer core contracts deployed. It also includes the [dora](https://github.com/ethpandaops/dora) and [blockscout](https://github.com/blockscout/blockscout) explorers.

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

For example configurations, check [`examples`](examples/). For more information on the config file format, check [Configuration](#configuration).

### Stop the devnet

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

## Local development

We have a Makefile for some of the usual tasks.

### Starting the devnet

This registers the devnet with Kurtosis, and runs it.
The command can be run multiple times, each one updating the devnet configuration.

```sh
make start_devnet
```

### Stopping the devnet

This stops the devnet without removing containers and file artifacts.

```sh
make stop_devnet
```

### Removing the devnet

This stops the devnet, removing containers and file artifacts.

```sh
make clean_devnet
```

### Formatting files

To format the Starlark scripts, run:

```sh
make format
```

### Starting an example

Some of the targets run with the configurations under [examples](./examples/).

```sh
# https://github.com/Layr-Labs/incredible-squaring-avs
make start_incredible_squaring
# https://github.com/Layr-Labs/hello-world-avs
make start_hello_world
```

## Configuration

An example (non-functional) configuration is:

```yaml
# Lists the contracts to deploy
deployments:
    # The name of the contract group
  - name: deployment-name
    # The repo to fetch the contracts from
    repo: "https://github.com/some-org/some-repo.git"
    # The commit/branch/tag to use
    ref: "d05341ef33e5853fd3ecef831ae4dcfbf29c5299"
    # The path to the foundry project inside the repo
    contracts_path: "contracts/"
    # The path to the deployer script (may include the contract name after ':')
    script: script/deploy/devnet/M2_Deploy_From_Scratch.s.sol:Deployer_M2
    # Extra args passed on to `forge script`
    extra_args: --sig 'run(string memory configFile)' -- deploy_from_scratch.config.json
    # Input files to embed into the repo
    input:
      # Key: destination to insert the files in
      # Value: name of the artifact containing the files
      script/configs/devnet/: eigenlayer_deployment_input
      # Multiple artifacts can be specified and all artifact files will be stored
      # in the directory
      some/other/dir/: 
        - file_a
        - file_b
    # Output files to store after execution
    output:
      # Key: name of the new artifact
      # Value: path to the file to store in the artifact
      eigenlayer_addresses: "script/output/devnet/M2_from_scratch_deployment_data.json"
      # You can also specify a new name for the file before storing it
      eigenlayer_addresses_renamed:
        # Same as before
        path: "script/output/devnet/M2_from_scratch_deployment_data.json"
        # The new name to give to the file
        rename: "eigenlayer_deployment_output.json"

# Lists the services to start after the contracts are deployed
services:
    # Name for the service
  - name: "aggregator"
    # The docker image to use
    image: "ghcr.io/layr-labs/incredible-squaring/aggregator/cmd/main.go:latest"
    # The ports to expose on the container
    ports:
      # The key is a name for the port
      port_name:
        # Port number
        number: 8090
        # Port transport protocol: TCP, UDP
        transport_protocol: "TCP"
        # Application protocol: HTTP, etc.
        application_protocol: "http"
        # Timeout before failing deployment. `null` can be used to disable this.
        # Default: 15s
        wait: "10s"
    # [Optional] Ethereum address the service will use. Funds will be deposited to it at startup.
    # NOTE: this option will be removed in the future
    address: "0xa0Ee7A142d267C1f36714E4a8F75612F20a79720"
    # Input files to embed into the repo
    # Same as in `deployments`
    input:
      key: value
    # Used to specify environment variables to pass to the image
    env:
      # Key: env variable name
      # Value: env variable's value
      key: value
      # Values starting with `$` can be used to retrieve context information
      # This example expands to the `ecdsa_keys` keystore's password
      ECDSA_KEY_PASSWORD: $keys.ecdsa_keys.password
    # Command to use when running the docker image
    cmd: ["some", "option", "here"]

# Lists the keys to be generated at startup
keys:
    # Name for the keys
  - name: "ecdsa_keys"
    # Type of keys: bls, ecdsa
    type: "ecdsa"

# Lists artifacts to be generated at startup
artifacts:
  # Artifact name
  eigenlayer_deployment_input:
    # Data from other artifacts to use in the generation
    additional_data:
      # Artifact name to fetch data from
      artifact_name:
        # Key: name of the variable to populate
        # Value: JSONPath to the data
        # NOTE: this assumes that the data inside the artifact is a single JSON file
        some_variable: ".field1.foo[0]"

    # List of files to store inside the artifact
    files:
      # Key: file name
      # Value: a string to be the file's contents.
      # The string is assumed to be a Go template
      # (see https://pkg.go.dev/text/template for more information).
      # There are also some dynamically populated fields like 'deployer_address'
      someconfig.config.json: |
        {
          "a": 5,
          "someVariable": {{.some_variable}},
          "deployerAddress": {{.deployer_address}}
        }

# Args to pass on to ethereum-package.
# See https://github.com/ethpandaops/ethereum-package for more information
ethereum_package:
  participants:
    - el_type: reth
      cl_type: teku
  additional_services:
    - blockscout
```
