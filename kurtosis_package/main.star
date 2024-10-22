# Import remote code from another package using an absolute import
ethereum_package = import_module("github.com/ethpandaops/ethereum-package/main.star")
service_utils = import_module("./service_utils.star")
shared_utils = import_module("./shared_utils.star")
contract_deployer = import_module("./contract_deployer.star")
keystore = import_module("./keystore.star")


def run(plan, args={}):
    # Run the Ethereum package first
    ethereum_args = args.get("ethereum_package", {})
    ethereum_output = ethereum_package.run(plan, ethereum_args)

    el_context = ethereum_output.all_participants[0].el_context
    http_rpc_url = el_context.rpc_http_url
    ws_url = el_context.ws_url

    pre_funded_account = ethereum_output.pre_funded_accounts[0]
    private_key = pre_funded_account.private_key
    deployer_address = pre_funded_account.address

    plan.print(
        "\n".join(
            [
                "Data used for deployment:",
                " rpc: {} (docker internal)".format(http_rpc_url),
                " deployer private key: 0x{}".format(private_key),
                " deployer address: {}".format(deployer_address),
            ]
        )
    )

    # Default to an empty dict
    args["artifacts"] = args.get("artifacts", {})
    data = {
        "HttpRpcUrl": http_rpc_url,
        "WsUrl": ws_url,
        "DeployerPrivateKey": "0x" + private_key,
        "DeployerAddress": deployer_address,
    }

    context = struct(
        artifacts=args["artifacts"],
        services={},
        ethereum=ethereum_output,
        data=data,
        passwords={},
    )

    keystores = args.get("keystores", [])
    keystore.generate_all_keystores(plan, context, keystores)

    deployments = args.get("deployments", [])

    for deployment in deployments:
        contract_deployer.deploy(plan, context, deployment)

    service_specs = args.get("services", [])

    for service in service_specs:
        service_utils.add_service(plan, service, context)

    return ethereum_output
