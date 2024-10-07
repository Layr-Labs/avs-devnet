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

    setup_operator_config(plan, http_rpc_url)

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
                "/usr/src/app/config-files/": Directory(
                    artifact_names = ["operator-updated-config", "operator_bls_keystore", "operator_ecdsa_keystore"],
                ),
            },
            cmd=["--config", "/usr/src/app/config-files/operator-config.yaml"]
        ),

    )
    aggregator_config = plan.upload_files(
        src="./aggregator.yaml",
        name="aggregator-config",
    )
    operator_ecdsa_keystore = plan.upload_files(
        src="./test.ecdsa.key.json",
        name="aggregator_ecdsa_keystore",
    )

    aggregator = plan.add_service(
        name = "ics-aggregator",
        config = ServiceConfig(
            image = "ghcr.io/layr-labs/incredible-squaring/aggregator/cmd/main.go:latest",
            ports = {
                "rpc": PortSpec(
                    number = 9001,
                    transport_protocol = "TCP",
                    application_protocol = "http",
                    wait = "5s",
                ),
            },
            files = {
                "/usr/src/app/config-files/": Directory(
                    artifact_names = ["aggregator-config", "aggregator_ecdsa_keystore", "eigenlayer_addresses"],
                ),
            },
            cmd=["--ecdsa-private-key", "/usr/src/app/config-files/aggregator.yaml", "--credible-squaring-deployment", "/usr/src/app/config-files/M2_from_scratch_deployment_data.json"]
        ),

    )


    # return ethereum_output


def setup_operator_config(plan, http_rpc_url):
    operator_config = plan.upload_files(
        src="./operator-config.yaml",
        name="operator-config",
    )

    operator_bls_keystore = plan.upload_files(
        src="./test.bls.key.json",
        name="operator_bls_keystore",
    )
    operator_ecdsa_keystore = plan.upload_files(
        src="./test.ecdsa.key.json",
        name="operator_ecdsa_keystore",
    )
    
    eigenlayer_addresses = plan.get_files_artifact(
        name = "eigenlayer_addresses",
        description = "gets you an artifact",
    )
    # get avsDirectory
    result = plan.run_sh(
        image="badouralix/curl-jq",
        run="jq -j .addresses.avsDirectory /usr/src/app/config-files/M2_from_scratch_deployment_data.json",
        files={
            "/usr/src/app/config-files/": eigenlayer_addresses,
        },
        wait="1s",
    )
    avs_directory = result.output
    
    # get strategyManager
    result = plan.run_sh(
        image="badouralix/curl-jq",
        run="jq -j .addresses.strategyManager /usr/src/app/config-files/M2_from_scratch_deployment_data.json",
        files={
            "/usr/src/app/config-files/": eigenlayer_addresses,
        },
        wait="1s",
    )
    strategy_manager = result.output

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
        "AvsRegistryCoordinatorAddress": avs_directory,
        "OperatorStateRetrieverAddress": strategy_manager,
        "EthRpcUrl": http_rpc_url,
        "EthWsUrl": http_rpc_url,
        "EcdsaPrivateKeyStorePath": "/usr/src/app/config-files/test.ecdsa.key.json",
        "BlsPrivateKeyStorePath": "/usr/src/app/config-files/test.bls.key.json",
        "AggregatorServerIpPortAddress": "9",
        "EigenMetricsIpPortAddress": "10",
        "EnableMetrics": False,
        "NodeApiIpPortAddress": "12",
        "EnableNodeApi": False,
        "RegisterOperatorOnStartup": False,
        "TokenStrategyAddr": "15",
    }

    artifact_name = plan.render_templates(
        config = {
            "operator-config.yaml": struct(
                template="\n".join([
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
                    "token_strategy_addr: {{.TokenStrategyAddr}}"
                ]),
                data=template_data,
            ),
        },
        name = "operator-updated-config",
        description = "rendering a template"  
    )