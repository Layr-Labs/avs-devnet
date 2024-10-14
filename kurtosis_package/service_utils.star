def add_service(plan, service_args, context):
    name = service_args["name"]
    files = generate_service_files(plan, context, service_args.get("input", {}))
    address = service_args.get("address", None)

    if address != None:
        send_funds(plan, context, address)

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
    context.data["Service_" + name] = service.ip_address


def generate_service_files(plan, context, input_args):
    files = {}

    for path, artifact_names in input_args.items():
        if len(artifact_names) == 0:
            continue
        for artifact_name in artifact_names:
            if artifact_name not in context.artifacts:
                continue
            if context.artifacts[artifact_name].get("generated", False):
                continue
            generate_artifact(plan, context, artifact_name)
        files[path] = Directory(artifact_names=artifact_names)

    return files


def generate_artifact(plan, context, artifact_name):
    artifact_files = context.artifacts[artifact_name].get("files", {})
    additional_data = context.artifacts[artifact_name].get("additional_data", {}) or {}
    data = dict(context.data)
    for artifact, vars in additional_data.items():
        for varname, json_field in vars.items():
            data[varname] = read_json_artifact(plan, artifact, json_field)
    config = {}
    for name, template in artifact_files.items():
        config[name] = struct(template=template, data=data)
    plan.render_templates(
        config=config,
        name=artifact_name,
        description="Generating '{}'".format(artifact_name),
    )


def read_json_artifact(plan, artifact_name, json_field):
    input_dir = "/_input"
    # get registryCoordinator
    result = plan.run_sh(
        image="badouralix/curl-jq",
        run="jq -j {field} {input}/*.json".format(field=json_field, input=input_dir),
        files={input_dir: artifact_name},
        wait="1s",
    )
    return result.output


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


# Taken from ethereum-package
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


def generate_env_vars(context, env_vars):
    return {
        env_var_name: expand(context, env_var_value)
        for env_var_name, env_var_value in env_vars.items()
    }


def expand(context, value):
    if not value.startswith("$"):
        return value

    artifact = value[1:].rstrip("_password")
    return context.passwords[artifact]


def send_funds(plan, context, to):
    http_rpc_url = context.ethereum.all_participants[0].el_context.rpc_http_url
    funded_private_key = context.ethereum.pre_funded_accounts[0].private_key
    amount = "10ether"
    plan.run_sh(
        image="ghcr.io/foundry-rs/foundry:nightly-471e4ac317858b3419faaee58ade30c0671021e0",
        run="cast send --value "
        + amount
        + " --private-key "
        + funded_private_key
        + " --rpc-url "
        + http_rpc_url
        + " "
        + to,
        description="Depositing funds into the account '" + to + "'",
    )
