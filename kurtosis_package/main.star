# Import remote code from another package using an absolute import
ethereum_package = import_module("github.com/ethpandaops/ethereum-package/main.star")
service_utils = import_module("./service_utils.star")
shared_utils = import_module("./shared_utils.star")
contract_deployer = import_module("./contract_deployer.star")


def run(plan, args={}):
    # Run the Ethereum package first
    ethereum_args = args.get("ethereum_params", {})
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
    generate_keystores(plan, context, keystores)

    deployments = args.get("deployments", [])

    for deployment in deployments:
        contract_deployer.deploy(plan, context, deployment)

    service_specs = args.get("services", [])

    for service in service_specs:
        service_utils.add_service(plan, service, context)

    return ethereum_output


def generate_keystores(plan, context, keystores):
    if len(keystores) == 0:
        return

    generator_service = plan.add_service(
        "egnkey-service",
        config=ServiceConfig(
            image=ImageBuildSpec(
                image_name="egnkey",
                build_context_dir="./dockerfiles/",
                build_file="egnkey.Dockerfile",
            ),
            entrypoint=["sleep", "99999"],
            description="Spinning up EigenLayer key generator service",
        ),
    )

    for keystore in keystores:
        name = keystore["name"]
        key_type = keystore["type"]
        _, password = generate_key(plan, generator_service.name, key_type, name)

        if key_type == "ecdsa":
            address = shared_utils.read_json_artifact(plan, name, ".address")
            service_utils.send_funds(plan, context, address)

        context.passwords[name] = password

    plan.remove_service(generator_service.name)


def generate_key(plan, egnkey_service_name, key_type, artifact_name):
    tmp_dir = "/_tmp"
    output_dir = "/_output"

    cmd = "rm -rf {tmp} && mkdir -p {output} && egnkey generate --key-type {type} --num-keys 1 --output-dir {tmp} && mv {tmp}/keys/1.{type}.key.json {output} ; cat {tmp}/password.txt | tr -d '\n'".format(
        tmp=tmp_dir, output=output_dir, type=key_type
    )

    result = plan.exec(
        service_name=egnkey_service_name,
        recipe=ExecRecipe(command=["sh", "-c", cmd]),
        description="Generating " + key_type + " key",
    )
    password = result["output"]

    file_artifact = plan.store_service_files(
        service_name=egnkey_service_name,
        src=output_dir + "/1." + key_type + ".key.json",
        name=artifact_name,
        description="Storing " + key_type + " key",
    )
    return file_artifact, password
