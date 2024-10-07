# Import remote code from another package using an absolute import
ethereum_package = import_module("github.com/ethpandaops/ethereum-package/main.star")


def run(plan, args={}):
    ethereum_args = args.get("ethereum_args", {})
    ethereum_output = ethereum_package.run(plan, ethereum_args)

    
    el_context = ethereum_output.all_participants[0].el_context
    http_rpc_url = el_context.rpc_http_url
    private_key = ethereum_output.pre_funded_accounts[0].private_key

    deploy_config_file_artifact = plan.upload_files(
        src="./static_files/deploy_from_scratch.config.json",
        name="eigenlayer-deployment-input",
    )

    # TODO: arg
    eigenlayer_contracts_version = "v0.4.2-mainnet-pepe"
    plan.run_sh(
        # Nightly (2024-10-03)
        image="contract_deployer",
        run="forge script ./script/deploy/devnet/M2_Deploy_From_Scratch.s.sol:Deployer_M2 --rpc-url ${HTTP_RPC_URL}  --private-key 0x${PRIVATE_KEY} --broadcast --sig 'run(string memory configFile)' -- deploy_from_scratch.config.json",
        env_vars={
            "EIGENLAYER_CONTRACTS_VERSION": eigenlayer_contracts_version,
            "HTTP_RPC_URL": http_rpc_url,
            "PRIVATE_KEY": private_key,
        },
        files={
            "/app/contracts/script/configs/devnet/": deploy_config_file_artifact,
        },
        store=[
            StoreSpec(src = "/app/contracts/script/output/devnet/M2_from_scratch_deployment_data.json", name = "eigenlayer_addresses")
        ],
        wait="15s",
    )

    operator_config = plan.upload_files(
        src="./operator-config.yaml",
        name="operator-config",
    )

    operator = plan.add_service(
        name = "ics-operator",
        config = ServiceConfig(
            image = "ghcr.io/layr-labs/incredible-squaring/operator/cmd/main.go:latest",
            ports = {
                "rpc": PortSpec(
                    number = 9000,
                    transport_protocol = "TCP",
                    application_protocol = "http",
                    wait = "5s",
                ),
            },
            files = {
                "/usr/src/app/config-files/": operator_config
            },
            cmd=["--config", "/usr/src/app/config-files/operator-config.yaml"]
        ),

    )
    
    return ethereum_output
