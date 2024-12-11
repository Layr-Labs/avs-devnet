# EigenLayer AVS Devnet

*AvsDevnet* is a CLI tool and library to start highly customizable local devnets.
The CLI tool is used in place of bash scripts for end-to-end testing and local development.
The library, on the other hand, is commonly used in place of mocks for automated testing of specific situations.

> [!WARNING]
> Currently, only the [Kurtosis package](./kurtosis_package/) and [CLI](./cli/) are available.
> Future versions may include the testing library.

## Dependencies

Since the Devnet is implemented as a Kurtosis package, we require Kurtosis to be installed.
For how to install it, you can check [here](https://docs.kurtosis.com/install/).
As part of that, you'll also need to install Docker.

For deploying local contracts, [foundry needs to be installed](https://book.getfoundry.sh/getting-started/installation).
Also, only contracts inside foundry projects are supported as of now.

For development, we require [the `go` toolchain to be installed](https://go.dev/doc/install).

> [!IMPORTANT]  
> To be able to [install the CLI via `go`](#installation), you'll need to add `$HOME/go/bin` to your `PATH`.
> You can do so by adding to your `~/.bashrc` (or the equivalent for your shell) the following line:
>
> ```bash
> export PATH="$PATH:$HOME/go/bin"
> ```

## Installation

To build and install the CLI locally, run:

```sh
make deps      # installs dependencies
make install   # installs the project
```

## How to Use

### Creating a devnet config

This will create a new devnet config.
By default it's stored as `devnet.yaml`, but another name can be passed as parameter.

```sh
devnet init
```

The default configuration deploys EigenLayer with a single strategy and operator.
It also starts up a [blockscout explorer](https://github.com/blockscout/blockscout).

### Starting the devnet

This will start a devnet according to the configuration inside `devnet.yaml`.
Another file name can be specified as the first parameter.

```sh
devnet start
```

Note that only one devnet per file name can be running at the same time.
Trying to start another one (or the same one more than once) will fail.

### Stopping the devnet

This will stop the devnet according to the configuration inside `devnet.yaml`.
Another file name can be specified as the first parameter.

```sh
devnet stop
```

### Fetching the address of a contract

This will output the address of the deployed contract named `delegation`, from the artifact `eigenlayer_addresses`.
In the default configuration, this corresponds to the address of EigenLayer's `DelegationManager`.

```sh
$ devnet get-address eigenlayer_addresses:delegation
0x9f9F5Fd89ad648f2C000C954d8d9C87743243eC5
```

This works by parsing the JSON artifacts generated by the deployment scripts.
The command expects there to be a single file with a field called "addresses" under which addresses are listed.

More examples:

```sh
# print all addresses in eigenlayer_addresses artifact
$ devnet get-address eigenlayer_addresses:
{
  "addresses": {
    # ...
    "delegation": "0x9f9F5Fd89ad648f2C000C954d8d9C87743243eC5",
    # ...
    "strategies": {
      "MockETH": "0x2b45cD38B213Bbd3A1A848bf2467927c976877Cb"
    },
    # ...
  },
  # ...
}
# print the address under strategies -> MockETH
$ devnet get-address eigenlayer_addresses:strategies.MockETH
0x2b45cD38B213Bbd3A1A848bf2467927c976877Cb
# because we also search nested entries, the last one can be shortened to
$ devnet get-address eigenlayer_addresses:MockETH
0x2b45cD38B213Bbd3A1A848bf2467927c976877Cb
# by adding a . at the start, we disable the search function
$ devnet get-address eigenlayer_addresses:.MockETH  # this fails
Contract not found: eigenlayer_addresses:.MockETH
```

### Local development

Some fields in the config can be used to ease deployment of local projects.

The `repo` field in `deployments` accepts local paths.
This can be used when deployments should be done from locally available versions.

```yaml
deployments:
  - name: some-deployment
    repo: "foo/bar/baz"
```

The `build_context` field in `services`, if specified, allows the Devnet to automatically build docker images via `docker build`.
Images are built in the specified context, and tagged with the name specified in the `image` field.
If the build file is named something other than `Dockerfile`, or isn't located in the context, you can use `build_file` to specify the path.

```yaml
services:
  - name: my-service
    image: name-for-the-image
    build_context: path/to/context
    build_file: path/to/context/Dockerfile
```

For image builds requiring a custom command, you can use `build_cmd` to specify it.
This overrides the `build_context` and `build_file`.

```yaml
services:
  - name: my-service
    image: name-for-the-image
    build_cmd: "docker build . -t name-for-the-image && touch .finished"
```

### More Help

You can find the options for each command by appending `--help`:

```sh
$ devnet --help
NAME:
   devnet - start an AVS devnet

USAGE:
   devnet [global options] command [command options]

VERSION:
   development

COMMANDS:
   init         Initialize a devnet configuration file
   start        Start devnet from configuration file
   stop         Stop devnet from configuration file
   get-address  Get a devnet contract or EOA address
   get-ports    Get the published ports on the devnet
   help, h      Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
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
    # This can also be a local path (absolute or relative)
    # repo: ./foo/bar
    # The commit/branch/tag to use
    ref: "d05341ef33e5853fd3ecef831ae4dcfbf29c5299"
    # The path to the foundry project inside the repo
    contracts_path: "contracts/"
    # The path to the deployer script (may include the contract name after ':')
    script: script/deploy/devnet/M2_Deploy_From_Scratch.s.sol:Deployer_M2
    # Extra args passed on to `forge script`
    extra_args: --sig 'run(string memory configFile)' -- deploy_from_scratch.config.json
    # Verify with local blockscout explorer (default: false)
    verify: true
    # Environment variables to set for deployment
    env:
      # Key: env variable name
      # Value: env variable's value
      key: value
      # Values inside double brackets '{{ }}' are expanded at runtime according to Go template syntax
      PRIVATE_KEY: "{{.deployer_private_key}}"
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
    # Specifies addresses to extract from output artifacts
    addresses:
      # Key: name of the address
      # Value: `<artifact-name>:<jq-filter-to-apply>`, same syntax as `devnet get-address`
      my_contract: "eigenlayer_addresses:.addresses.avsDirectoryImplementation"

    # Available types: eigenlayer
    # This autofills some of the other options, and allows access
    # to additional arguments
  - type: eigenlayer
    # Same as before
    ref: v0.4.2-mainnet-pepe
    # The strategies to deploy, all of them backed by the same mocked token
    strategies:
      # The strategy name
      - MockETH
    # The operators to register in EigenLayer
    operators:
        # The name of the operator
      - name: operator1
        # The keys
        keys: ecdsa_keys
        # The strategies to deposit shares in
        strategies:
          # strategy_name: number_of_tokens
          MockETH: 100000000000000000

# Lists the services to start after the contracts are deployed
services:
    # Name for the service
  - name: my-service
    # The docker image to use
    image: image-name
    # Local images are built automatically when specifying `build_context`
    # Specifies the context for the image's dockerfile
    build_context: path/to/context
    # Optional. Used to override the default of "build_context/Dockerfile".
    build_file: path/to/context/Dockerfile
    # Specifies a custom command for building the image.
    # This overrides the `build_context` and `build_file` options.
    build_cmd: "docker build . -t image-name && touch somefile.txt"
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
    # Input files to embed into the repo
    # Same as in `deployments`
    input:
      key: value
    # Used to specify environment variables to pass to the image
    env:
      # Key: env variable name
      # Value: env variable's value
      key: value
      # Values inside double brackets '{{ }}' (templates) are expanded
      # at runtime according to Go template syntax.
      # This example expands to the `ecdsa_keys` keystore's password
      ECDSA_KEY_PASSWORD: "{{.keys.ecdsa_keys.password}}"
    # Command to use when running the docker image
    # Options may contain templates
    cmd: ["some", "option", "here", "{{.keys.ecdsa_keys.address}}"]

# Lists the keys to be generated at startup
keys:
    # Name for the keys
  - name: "ecdsa_keys"
    # Type of keys: bls, ecdsa
    type: "ecdsa"
    # Key details will be dynamically generated unless specified
    # Address of the precomputed key
    address: "0xdeadbeef"
    # Private key of the precomputed key
    private_key: "0xdeadbeef"

# Lists artifacts to be generated at startup
artifacts:
  # Artifact name
  eigenlayer_deployment_input:
    # Data from other artifacts to use in the generation
    additional_data:
      # Artifact name to fetch data from
      artifact_name:
        # Key: name of the variable to populate
        # Value: jq filter to extract the data
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
          "deployerAddress": {{.deployer_address}},
          "avsDirectory": {{.addresses.EigenLayer.avsDirectory}},
          "contractAddress": {{(index .addresses "deployment-name").my_contract}}
        }

# Args to pass on to ethereum-package.
# See https://github.com/ethpandaops/ethereum-package for more information
ethereum_package:
  additional_services:
    - blockscout
```

### Context object

The *Context* is used when expanding templates.
What follows is a list of its contents.

#### `.http_rpc_url`

The URL of the HTTP-RPC exposed on the first node of the underlying devnet.

Example value: `http://172.16.0.9:8545`

#### `.ws_rpc_url`

The URL of the WebSocket-RPC exposed on the first node of the underlying devnet.

Example value: `ws://172.16.0.9:8546`

#### `.deployer_private_key`

The ECDSA private key used when deploying contracts.

Example value: `0xbcdf20249abf0ed6d944c0288fad489e33f66b3960d9e6229c1cd214ed3bbe31`

#### `.deployer_address`

The address used to deploy contracts.

Example value: `0x8943545177806ED17B9F23F0a21ee5948eCaa776`

#### `.addresses.<deployment-name>.<contract-name>`

The address of the contract `<contract-name>` from deployment `<deployment-name>`.
Note that this requires the address to be declared before the template expansion.

Example value: `{{.addresses.MyAvs.serviceManager}}` expands to `0x89a37F5cd42162B56DE8A48bDe38A6E97C965675`

#### `.services.<service-name>.ip_address`

The IP address of the service `<service-name>`.
Note that this requires the service to be started before the template expansion.

Example value: `{{.services.aggregator.ip_address}}` expands to `172.16.0.70`

#### `.keys.<key-name>.address`

The Ethereum address associated to the key named `<key-name>`.
Only ECDSA keys have this property.

Example value: `0x0d7597aedfa6b73f3aac93ecfcf5abcfbcc5cd40`

#### `.keys.<key-name>.private_key`

The private key of the key named `<key-name>`.

Example value:

- ECDSA: `0xe314a391f6e0128c35573c9157baedd8381350e4efdc7e73509849a8e0b73f32`
- BLS: `11311926940818870267862834934784331525396505743635597567466859068964031983193`

#### `.keys.<key-name>.password`

The password to the keystore for the key named `<key-name>`.
Only dynamically generated keys have this property.

Example value: `jR07sE6zmoIElmwjsf7m`

## Kurtosis package

For how to use the Kurtosis package or interact with the devnet via Kurtosis CLI, see the documentation available in [`docs/kurtosis_package.md`](./docs/kurtosis_package.md).

## Contributing

We have a Makefile for some of the usual tasks.
Run `make help` for more info.

## Disclaimer

🚧 AvsDevnet is under active development and has not been audited. AvsDevnet is rapidly being upgraded, features may be added, removed or otherwise improved or modified and interfaces will have breaking changes. AvsDevnet should be used only for testing purposes and not in production. AvsDevnet is provided "as is" and Eigen Labs, Inc. does not guarantee its functionality or provide support for its use in production. 🚧
