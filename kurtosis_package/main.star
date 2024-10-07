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
                "/usr/src/app/config-files/": operator_config
            },
            cmd=["--config", "/usr/src/app/config-files/operator-config.yaml"]
        ),

    )

    return ethereum_output
