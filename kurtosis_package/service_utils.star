shared_utils = import_module("./shared_utils.star")


def add_service(plan, service_args, context):
    name = service_args["name"]
    files = shared_utils.generate_input_files(
        plan, context, service_args.get("input", {})
    )
    address = service_args.get("address", None)

    if address != None:
        shared_utils.send_funds(plan, context, address)

    ports = generate_port_specs(service_args.get("ports", {}))
    env_vars = generate_env_vars(context, service_args.get("env", {}))
    config = ServiceConfig(
        image=service_args["image"],
        ports=ports,
        files=files,
        env_vars=env_vars,
        cmd=service_args.get("cmd", []),
    )
    plan.print(config)
    service = plan.add_service(
        name=name,
        config=config,
    )
    context.services[name] = service
    # TODO: expose more service information
    context.data["services"][name] = service


def generate_port_specs(ports):
    return {
        port_name: new_port_spec(port_spec) for port_name, port_spec in ports.items()
    }


def new_port_spec(port_spec_args):
    number = port_spec_args["number"]
    transport_protocol = port_spec_args["transport_protocol"]
    application_protocol = port_spec_args.get("application_protocol", None)
    wait = port_spec_args.get("wait", "15s")

    return PortSpec(
        number=number,
        transport_protocol=transport_protocol,
        application_protocol=application_protocol,
        wait=wait,
    )


def generate_env_vars(context, env_vars):
    return {
        env_var_name: expand(context, env_var_value)
        for env_var_name, env_var_value in env_vars.items()
    }


def expand(context, value):
    if not value.startswith("$"):
        return value

    # $RPC_URL expands to the RPC URL of the first Ethereum node
    # TODO: store this in some other place
    if value.startswith("$RPC_URL"):
        return context.ethereum.all_participants[0].el_context.rpc_http_url

    # $name.password expands to the password of the keystore named `name`
    artifact = value[1:].rstrip(".password")
    return context.passwords[artifact]
