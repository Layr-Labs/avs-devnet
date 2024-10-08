# Import remote code from another package using an absolute import
ethereum_package = import_module("github.com/ethpandaops/ethereum-package/main.star")


def gen_deployer_img(repo, ref, path):
    name = repo.rstrip(".git").split("/")[-1]
    ref_name = ref.replace("/", "_")
    # Generate a unique identifier for the image
    uid = hash(str(repo + chr(0) + ref + chr(0) + path)) % 1000000
    return ImageBuildSpec(
        image_name="{name}_{ref}_deployer_{uid}".format(
            name=name, ref=ref_name, uid=uid
        ),
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

    # TODO: generalize this for any app
    eigenlayer_repo = args.get(
        "eigenlayer_repo", "https://github.com/Layr-Labs/incredible-squaring-avs.git"
    )
    eigenlayer_ref = args.get("eigenlayer_ref", "master")
    eigenlayer_path = args.get(
        "eigenlayer_path",
        "contracts/lib/eigenlayer-middleware/lib/eigenlayer-contracts",
    )

    avs_repo = args.get(
        "avs_repo", "https://github.com/Layr-Labs/incredible-squaring-avs.git"
    )
    avs_ref = args.get("avs_ref", "master")
    avs_path = args.get("avs_path", "contracts")

    chain_id = ethereum_args.get("network_params", {"network_id": 3151908})[
        "network_id"
    ]

    el_context = ethereum_output.all_participants[0].el_context
    http_rpc_url = el_context.rpc_http_url
    ws_url = el_context.ws_url

    pre_funded_account = ethereum_output.pre_funded_accounts[0]
    private_key = pre_funded_account.private_key
    eth_address = pre_funded_account.address

    el_config_template = read_file(
        "static_files/deploy_from_scratch.config.json.template"
    )

    el_config_data = {
        "OperationsMultisig": str(eth_address),
        "PauserMultisig": str(eth_address),
        "ExecutorMultisig": str(eth_address),
    }

    deploy_config_file_artifact = plan.render_templates(
        config={
            "deploy_from_scratch.config.json": struct(
                template=el_config_template,
                data=el_config_data,
            )
        },
        name="eigenlayer-deployment-input",
        description="Generating EigenLayer deployment configuration file",
    )

    eigenlayer_deployer_img = gen_deployer_img(
        eigenlayer_repo, eigenlayer_ref, eigenlayer_path
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

    # NOTE: we're using `;` because using `&&` after `--` fails to execute following commands
    cmd = "forge script ./script/deploy/devnet/M2_Deploy_From_Scratch.s.sol:Deployer_M2 \
            --rpc-url ${HTTP_RPC_URL}  --private-key 0x${PRIVATE_KEY} --broadcast --sig 'run(string memory configFile)' \
            -- deploy_from_scratch.config.json  ; \
        mv ./script/output/devnet/M2_from_scratch_deployment_data.json /eigenlayer_deployment_output.json"

    # Deploy the EigenLayer contracts
    result = plan.run_sh(
        image=eigenlayer_deployer_img,
        run=cmd,
        env_vars={
            "HTTP_RPC_URL": http_rpc_url,
            "PRIVATE_KEY": private_key,
        },
        files={
            "/app/{}/script/configs/devnet/".format(
                eigenlayer_path
            ): deploy_config_file_artifact,
        },
        store=[
            StoreSpec(
                src="/eigenlayer_deployment_output.json",
                name="eigenlayer_addresses",
            )
        ],
        description="Deploying EigenLayer contracts",
    )
    eigenlayer_deployment_file = result.files_artifacts[0]

    # If AVS wasn't provided, we skip setting it up
    if avs_repo == None or avs_path == None or avs_ref == None:
        return ethereum_output

    ics_deployer_img = gen_deployer_img(
        avs_repo,
        avs_ref,
        avs_path,
    )

    output_dir = "/app/{}/script/output/{}/".format(avs_path, chain_id)

    # Deploy the Incredible Squaring AVS contracts
    result = plan.run_sh(
        image=ics_deployer_img,
        run="forge script ./script/IncredibleSquaringDeployer.s.sol --rpc-url ${HTTP_RPC_URL}  --private-key 0x${PRIVATE_KEY} --broadcast -vvv",
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

    setup_operator_config(plan, http_rpc_url, ws_url)

    operator = plan.add_service(
        name="ics-operator",
        config=ServiceConfig(
            image="ghcr.io/layr-labs/incredible-squaring/operator/cmd/main.go:latest",
            ports={
                "rpc": PortSpec(
                    number=8545,
                    transport_protocol="TCP",
                    application_protocol="http",
                    wait=None,
                ),
            },
            files={
                "/usr/src/app/config-files/": Directory(
                    artifact_names=[
                        "operator-config",
                        "operator_bls_keystore",
                        "operator_ecdsa_keystore",
                    ],
                ),
            },
            cmd=["--config", "/usr/src/app/config-files/operator-config.yaml"],
        ),
    )

    return ethereum_output


def setup_operator_config(plan, http_rpc_url, ws_url):
    operator_bls_keystore = plan.upload_files(
        src="./test.bls.key.json",
        name="operator_bls_keystore",
    )
    operator_ecdsa_keystore = plan.upload_files(
        src="./test.ecdsa.key.json",
        name="operator_ecdsa_keystore",
    )

    avs_addresses = plan.get_files_artifact(
        name="avs_addresses",
        description="Getting AVS addresses file",
    )
    # get registryCoordinator
    result = plan.run_sh(
        image="badouralix/curl-jq",
        run="jq -j .addresses.registryCoordinator /usr/src/app/config-files/credible_squaring_avs_deployment_output.json",
        files={
            "/usr/src/app/config-files/": avs_addresses,
        },
        wait="1s",
    )
    registry_coordinator_address = result.output

    # get operatorStateRetriever
    result = plan.run_sh(
        image="badouralix/curl-jq",
        run="jq -j .addresses.operatorStateRetriever /usr/src/app/config-files/credible_squaring_avs_deployment_output.json",
        files={
            "/usr/src/app/config-files/": avs_addresses,
        },
        wait="1s",
    )
    operator_state_retriever = result.output

    # get tokenStrategy
    result = plan.run_sh(
        image="badouralix/curl-jq",
        run="jq -j .addresses.erc20MockStrategy /usr/src/app/config-files/credible_squaring_avs_deployment_output.json",
        files={
            "/usr/src/app/config-files/": avs_addresses,
        },
        wait="1s",
    )
    token_strategy = result.output

    # get operator address
    result = plan.run_sh(
        image="badouralix/curl-jq",
        run="jq -j .address /usr/src/app/config-files/test.ecdsa.key.json",
        files={
            "/usr/src/app/config-files/": operator_ecdsa_keystore,
        },
        wait="1s",
    )
    operator_address = result.output

    template_data = {
        "Production": False,
        "OperatorAddress": operator_address,
        "AvsRegistryCoordinatorAddress": registry_coordinator_address,
        "OperatorStateRetrieverAddress": operator_state_retriever,
        "EthRpcUrl": http_rpc_url,
        "EthWsUrl": ws_url,
        "EcdsaPrivateKeyStorePath": "/usr/src/app/config-files/test.ecdsa.key.json",
        "BlsPrivateKeyStorePath": "/usr/src/app/config-files/test.bls.key.json",
        "AggregatorServerIpPortAddress": "9",
        "EigenMetricsIpPortAddress": "10",
        "EnableMetrics": False,
        "NodeApiIpPortAddress": "12",
        "EnableNodeApi": False,
        "RegisterOperatorOnStartup": True,
        "TokenStrategyAddr": token_strategy,
    }

    artifact_name = plan.render_templates(
        config={
            "operator-config.yaml": struct(
                template="\n".join(
                    [
                        "production: {{.Production}}",
                        "operator_address: {{.OperatorAddress}}",
                        "avs_registry_coordinator_address: {{.AvsRegistryCoordinatorAddress}}",
                        "operator_state_retriever_address: {{.OperatorStateRetrieverAddress}}",
                        "eth_rpc_url: {{.EthRpcUrl}}",
                        "eth_ws_url: {{.EthWsUrl}}",
                        "ecdsa_private_key_store_path: {{.EcdsaPrivateKeyStorePath}}",
                        "bls_private_key_store_path: {{.BlsPrivateKeyStorePath}}",
                        "aggregator_server_ip_port_address: {{.AggregatorServerIpPortAddress}}",
                        "eigen_metrics_ip_port_address: {{.EigenMetricsIpPortAddress}}",
                        "enable_metrics: {{.EnableMetrics}}",
                        "node_api_ip_port_address: {{.NodeApiIpPortAddress}}",
                        "enable_node_api: {{.EnableNodeApi}}",
                        "register_operator_on_startup: {{.RegisterOperatorOnStartup}}",
                        "token_strategy_addr: {{.TokenStrategyAddr}}",
                    ]
                ),
                data=template_data,
            ),
        },
        name="operator-config",
        description="Generating operator configuration file",
    )
