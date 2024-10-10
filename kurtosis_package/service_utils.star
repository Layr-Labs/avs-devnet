def add_service(plan, service_args, ethereum_output):
    name = service_args["name"]
    files = generate_service_files(plan, service_args["input"])
    address = service_args.get("address", None)

    if address != None:
        http_rpc_url = ethereum_output.all_participants[0].el_context.rpc_http_url
        funded_private_key = ethereum_output.pre_funded_accounts[0].private_key
        plan.run_sh(
            image="ghcr.io/foundry-rs/foundry:nightly-471e4ac317858b3419faaee58ade30c0671021e0",
            run="cast send --value 10ether --private-key "
            + funded_private_key
            + " --rpc-url "
            + http_rpc_url
            + " "
            + address,
            description="Depositing funds into the account of service '{}'".format(
                name
            ),
        )

    ports = generate_port_specs(service_args["ports"])
    config = ServiceConfig(
        image=service_args["image"],
        ports=ports,
        public_ports=ports,
        files=files,
        cmd=service_args["cmd"],
    )
    plan.print(config)
    plan.add_service(
        name=name,
        config=config,
    )


def generate_service_files(plan, input_args):
    files = {}

    for path, artifact_names in input_args.items():
        if len(artifact_names) == 0:
            continue
        files[path] = Directory(artifact_names=artifact_names)

    return files


def generate_port_specs(ports):
    return {
        port_name: new_port_spec(
            port_spec["number"],
            port_spec["transport_protocol"],
            port_spec.get("application_protocol", None),
            port_spec.get("wait", None),
        )
        for port_name, port_spec in ports.items()
    }


def new_port_spec(
    number,
    transport_protocol,
    application_protocol=None,
    wait=None,
):
    if wait == None:
        return PortSpec(
            number=number,
            transport_protocol=transport_protocol,
            application_protocol=application_protocol or "",
        )

    return PortSpec(
        number=number,
        transport_protocol=transport_protocol,
        application_protocol=application_protocol,
        wait=wait,
    )
