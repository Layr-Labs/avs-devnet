# Import remote code from another package using an absolute import
ethereum_package = import_module("github.com/ethpandaops/ethereum-package/main.star")
service_utils = import_module("./service_utils.star")
shared_utils = import_module("./shared_utils.star")
contract_deployer = import_module("./contract_deployer.star")
keystore = import_module("./keystore.star")


def run(plan, args={}):
    ethereum_args = args.get("ethereum_package", {})
    args = parse_args(plan, args)

    # Run the Ethereum package first
    ethereum_output = ethereum_package.run(plan, ethereum_args)

    el_context = ethereum_output.all_participants[0].el_context

    if el_context.client_name == "geth":
        fail("Geth is not supported yet. See this issue for more details: https://github.com/Layr-Labs/avs-devnet/issues/22")

    http_rpc_url = el_context.rpc_http_url
    ws_url = el_context.ws_url

    pre_funded_account = ethereum_output.pre_funded_accounts[0]
    private_key = pre_funded_account.private_key
    deployer_address = pre_funded_account.address

    data = {
        "http_rpc_url": http_rpc_url,
        "ws_rpc_url": ws_url,
        "deployer_private_key": "0x" + private_key,
        "deployer_address": deployer_address,
    }
    plan.print("Initial data: " + json.indent(json.encode(data)))

    # Append fields that will be populated later
    data.update({"services": {}, "keystores": {}, "addresses": {}})

    context = struct(
        artifacts=args.artifacts,
        services={},
        ethereum=ethereum_output,
        data=data,
    )

    keystore.generate_all_keystores(plan, context, args.keystores)

    for deployment in args.deployments:
        contract_deployer.deploy(plan, context, deployment)

    for service in args.services:
        service_utils.add_service(plan, service, context)

    return context


def parse_args(plan, args):
    artifacts = args.get("artifacts", {})
    keystores = args.get("keystores", [])
    deployments = args.get("deployments", [])
    services = args.get("services", [])

    return struct(
        artifacts=artifacts,
        keystores=keystores,
        deployments=deployments,
        services=services,
    )
