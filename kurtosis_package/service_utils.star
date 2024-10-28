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
    env_vars = shared_utils.generate_env_vars(context, service_args.get("env", {}))
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
    # TODO: we could expose more service data here
    context.data["services"][name] = {"ip_address": service.ip_address}


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
