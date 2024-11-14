# Import remote code from another package using an absolute import
ethereum_package = import_module("github.com/ethpandaops/ethereum-package/main.star")
service_utils = import_module("./service_utils.star")
shared_utils = import_module("./shared_utils.star")
contract_deployer = import_module("./contract_deployer.star")
keys = import_module("./keys.star")


def run(plan, args={}):
    ethereum_args = parse_ethereum_package_args(plan, args)
    args = parse_args(plan, args)

    # Run the Ethereum package first
    ethereum_output = ethereum_package.run(plan, ethereum_args)

    el_context = ethereum_output.all_participants[0].el_context
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
    data.update({"services": {}, "keys": {}, "addresses": {}})

    context = struct(
        artifacts=args.artifacts,
        services={},
        ethereum=ethereum_output,
        data=data,
    )

    keys.generate_all_keys(plan, context, args.keys)

    for deployment in args.deployments:
        contract_deployer.deploy(plan, context, deployment)

    for service in args.services:
        service_utils.add_service(plan, service, context)

    return context


def parse_args(plan, args):
    artifacts = args.get("artifacts", {})
    keys = args.get("keys", [])
    deployments = args.get("deployments", [])
    services = args.get("services", [])

    return struct(
        artifacts=artifacts,
        keys=keys,
        deployments=deployments,
        services=services,
    )


def parse_ethereum_package_args(plan, args):
    ethereum_args = dict(args.get("ethereum_package", {}))
    participants = ethereum_args.get("participants", [{"el_type": "besu"}])

    if len(participants) == 0 or participants[0].get("el_type") != "besu":
        plan.print("WARNING: no 'besu' client as first participant. Adding one...")
        participants = [{"el_type": "besu"}] + participants

    ethereum_args["participants"] = participants
    return ethereum_args
