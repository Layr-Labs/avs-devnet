# Import remote code from another package using an absolute import
ethereum_package = import_module("github.com/ethpandaops/ethereum-package/main.star")

def run(plan, args={}):
    ethereum_args = args.get("ethereum_args", {})
    ethereum_output = ethereum_package.run(plan, ethereum_args)

    operator_config = plan.upload_files(
        src="./operator-config.yaml",
        name="operator-config",
    )

    operator = plan.add_service(
        name = "ics-operator",
        config = ServiceConfig(
            image = "ics-operator",
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
            entrypoint = ["operator", "--config", "/usr/src/app/config-files/operator-config.yaml"],
        ),

    )

    return ethereum_output
