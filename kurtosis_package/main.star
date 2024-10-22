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

    artifacts = args.get("artifacts", {})
    keystores = args.get("keystores", [])
    deployments = args.get("deployments", [])
    service_specs = args.get("services", [])

    data = {
        "http_rpc_url": http_rpc_url,
        "ws_rpc_url": ws_url,
        "deployer_private_key": "0x" + private_key,
        "deployer_address": deployer_address,
        "services": {},
        "keystores": {},
    }

    context = struct(
        artifacts=artifacts,
        services={},
        ethereum=ethereum_output,
        data=data,
    )

    keystore.generate_all_keystores(plan, context, keystores)

    for deployment in deployments:
        contract_deployer.deploy(plan, context, deployment)

    for service in service_specs:
        service_utils.add_service(plan, service, context)

    return ethereum_output
