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

    eigenlayer_deployer_img = ImageBuildSpec(
        image_name="eigenlayer_deployer",
        build_context_dir="./dockerfiles/",
        build_file="contract_deployer.dockerfile",
        build_args={
            "CONTRACTS_REPO": "https://github.com/Layr-Labs/eigenlayer-contracts.git",
            "CONTRACTS_REF": "v0.4.2-mainnet-pepe",
            "CONTRACTS_PATH": ".",
        },
    )

    plan.run_sh(
        image=eigenlayer_deployer_img,
        run="forge script ./script/deploy/devnet/M2_Deploy_From_Scratch.s.sol:Deployer_M2 --rpc-url ${HTTP_RPC_URL}  --private-key 0x${PRIVATE_KEY} --broadcast --sig 'run(string memory configFile)' -- deploy_from_scratch.config.json",
        env_vars={
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
        description="Deploying EigenLayer contracts",
    )
    return ethereum_output
