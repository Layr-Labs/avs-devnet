# Import remote code from another package using an absolute import
ethereum_package = import_module("github.com/ethpandaops/ethereum-package/main.star")


def run(plan, args={}):
    ethereum_args = args.get("ethereum_args", {})
    ethereum_output = ethereum_package.run(plan, ethereum_args)
    http_rpc_url = ethereum_output.all_participants[0].el_context.rpc_http_url
    private_key = ethereum_output.pre_funded_accounts[0].private_key

    deploy_config_file_artifact = plan.upload_files(
        src="./static_files/deploy_from_scratch.config.json",
        name="eigenlayer-deployment-input",
    )
    # TODO: arg
    eigenlayer_contracts_version = "v0.4.2-mainnet-pepe"
    plan.run_sh(
        # Nightly (2024-10-03)
        image="ghcr.io/foundry-rs/foundry:nightly-471e4ac317858b3419faaee58ade30c0671021e0",
        run="mkdir /_contracts/ && cd /_contracts/ && git clone https://github.com/Layr-Labs/eigenlayer-contracts.git . --single-branch --branch ${EIGENLAYER_CONTRACTS_VERSION} --depth 1 --shallow-submodules --recurse-submodules && forge build && forge script ./script/deploy/local/Deploy_From_Scratch.s.sol:DeployFromScratch --rpc-url ${HTTP_RPC_URL}  --private-key 0x${PRIVATE_KEY} --broadcast --slow --sig 'run(string memory configFile)' -- /local/deploy_from_scratch.config.json",
        env_vars={
            "EIGENLAYER_CONTRACTS_VERSION": eigenlayer_contracts_version,
            "HTTP_RPC_URL": http_rpc_url,
            "PRIVATE_KEY": private_key,
        },
        files={
            "/local/": deploy_config_file_artifact,
        },
        wait=None,
    )
    return ethereum_output
