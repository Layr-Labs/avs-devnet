# EigenLayer AVS Devnet

*AvsDevnet* is a library and CLI tool to start local devnets with specific operator states.
We expect the library to be commonly used in place of mocks for automated testing of specific situations.
The CLI tool, on the other hand, should be used in place of anvil-like solutions for end-to-end testing.

## Features

### One line devnet setup

Currently, to have a local devnet with EigenLayer contracts deployed, we need to deploy them manually or build our own scripts.
This also includes deploying all of our AVS contracts.
With AvsDevnet we could make this as simple as a one line command.

### Extensively configurable

By having lots of tuning parameters for operators we can simulate complex situations.
We’re going to start operator registration and stakes setup only, but a lot of this could be extended in the future.

### Usable as a testing library

Being able to use it on unit tests will make automated testing easier.
With this, users won’t need to run complex setups before their tests.
They can just use the library and set the initial required state.

## Dependencies

Since the Devnet is implemented as a Kurtosis package, we require Kurtosis to be installed.
For how to install it, you can check [here](https://docs.kurtosis.com/install/).

## How to Run

To run it without cloning the repo, just use:

```sh
kurtosis run github.com/Layr-Labs/avs-devnet --enclave devnet --args-file github.com/kurtosis_package/devnet_params.yaml
```

> [!WARNING]  
> Since `Layr-Labs/avs-devnet` is a private repository, you'll need to login with `kurtosis github login` to access it.

We also have a Makefile with some targets for usual tasks.

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
      script/configs/devnet/: eigenlayer-deployment-input
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
      # Key: variable name
      # Value: variable's value
      key: value
      # These special values can be used to retrieve the password of a generated keystore
      # Syntax is $<keystore_name>.password
      ECDSA_KEY_PASSWORD: $ecdsa_keystore.password
    # Command to use when running the docker image
    cmd: ["some", "option", "here"]

# Lists the keystores to be generated at startup
keystores:
    # Name for the keystore
  - name: "ecdsa_keystore"
    # Type of keystore: bls, ecdsa
    type: "ecdsa"

# Lists artifacts to be generated at startup
artifacts:
  # Artifact name
  eigenlayer-deployment-input:
    # Data from other artifacts to use in the generation
    additional_data:
      # Artifact name to fetch data from
      artifact_name:
        # Key: name of the variable to populate
        # Value: JSONPath to the data
        # NOTE: this assumes that the data inside the artifact is a single JSON file
        SomeVariable: ".field1.foo[0]"

    # List of files to store inside the artifact
    files:
      # Key: file name
      # Value: a string to be the file's contents.
      # The string is assumed to be a Go template
      # (see https://pkg.go.dev/text/template for more information).
      someconfig.config.json: |
        {
          "a": 5,
          "someVariable": {{.SomeVariable}}
        }

# Args to pass on to ethereum-package.
# See https://github.com/ethpandaops/ethereum-package for more information
ethereum_params:
  participants:
    - el_type: reth
      cl_type: teku
  additional_services:
    - blockscout
    - dora
```
