# Import remote code from another package using an absolute import
ethereum_package = import_module("github.com/ethpandaops/ethereum-package/main.star")


def gen_deployer_img(repo, ref, path="."):
    name = repo.rstrip(".git").split("/")[-1]
    ref_name = ref.replace("/", "_")
    return ImageBuildSpec(
        image_name="{name}_{ref}_deployer".format(name=name, ref=ref_name),
        build_context_dir="./dockerfiles/",
        build_file="contract_deployer.dockerfile",
        build_args={
            "CONTRACTS_REPO": repo,
            "CONTRACTS_REF": ref,
            "CONTRACTS_PATH": path,
        },
    )


def run(plan, args={}):
    # Run the Ethereum package first
    ethereum_args = args.get("ethereum_params", {})
    ethereum_output = ethereum_package.run(plan, ethereum_args)

    chain_id = ethereum_args.get("network_params", {"network_id": 3151908})[
        "network_id"
    ]

    el_context = ethereum_output.all_participants[0].el_context
    http_rpc_url = el_context.rpc_http_url
    private_key = ethereum_output.pre_funded_accounts[0].private_key

    deploy_config_file_artifact = plan.upload_files(
        src="./static_files/deploy_from_scratch.config.json",
        name="eigenlayer-deployment-input",
        description="Uploading EigenLayer deployment configuration file",
    )

    eigenlayer_deployer_img = gen_deployer_img(
        "https://github.com/Layr-Labs/eigenlayer-contracts.git", "v0.4.2-mainnet-pepe"
    )

    plan.print(
        "\n".join(
            [
                "Data used for deployment:",
                " rpc: {} (docker internal)".format(http_rpc_url),
                " private key: 0x{}".format(private_key),
            ]
        )
    )

    # Deploy the EigenLayer contracts
    result = plan.run_sh(
        image=eigenlayer_deployer_img,
        run=" && ".join(
            [
                "forge script ./script/deploy/devnet/M2_Deploy_From_Scratch.s.sol:Deployer_M2 --rpc-url ${HTTP_RPC_URL}  --private-key 0x${PRIVATE_KEY} --broadcast --sig 'run(string memory configFile)' -- deploy_from_scratch.config.json",
                "mv /app/script/output/devnet/M2_from_scratch_deployment_data.json /app/script/output/devnet/eigenlayer_deployment_output.json",
            ]
        ),
        env_vars={
            "HTTP_RPC_URL": http_rpc_url,
            "PRIVATE_KEY": private_key,
        },
        files={
            "/app/script/configs/devnet/": deploy_config_file_artifact,
        },
        store=[
            StoreSpec(
                src="/app/script/output/devnet/eigenlayer_deployment_output.json",
                name="eigenlayer_addresses",
            )
        ],
        description="Deploying EigenLayer contracts",
    )
    eigenlayer_deployment_file = result.files_artifacts[0]

    ics_deployer_img = gen_deployer_img(
        "https://github.com/Layr-Labs/incredible-squaring-avs.git",
        "master",
        path="./contracts",
    )

    output_dir = "/app/contracts/script/output/{}/".format(chain_id)

    # Deploy the Incredible Squaring AVS contracts
    result = plan.run_sh(
        image=ics_deployer_img,
        run="forge script ./script/IncredibleSquaringDeployer.s.sol --rpc-url ${HTTP_RPC_URL}  --private-key 0x${PRIVATE_KEY} --broadcast -vvvv",
        env_vars={
            "HTTP_RPC_URL": http_rpc_url,
            "PRIVATE_KEY": private_key,
        },
        files={
            output_dir: eigenlayer_deployment_file,
        },
        store=[
            StoreSpec(
                src=output_dir + "credible_squaring_avs_deployment_output.json",
                name="avs_addresses",
            )
        ],
        description="Deploying Incredible Squaring contracts",
    )
    return ethereum_output
