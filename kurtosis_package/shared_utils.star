def generate_artifacts(plan, context, artifacts):
    for artifact_name in artifacts:
        generate_artifact(plan, context, artifact_name)


def generate_artifact(plan, context, artifact_name):
    if artifact_name not in context.artifacts:
        return artifact_name
    if context.artifacts[artifact_name].get("generated", False):
        return artifact_name
    artifact_files = context.artifacts[artifact_name].get("files", {})
    additional_data = context.artifacts[artifact_name].get("additional_data", {})
    data = dict(context.data)
    for artifact, vars in additional_data.items():
        for varname, json_field in vars.items():
            data[varname] = read_json_artifact(plan, artifact, json_field)
    config = {}
    for name, template in artifact_files.items():
        config[name] = struct(template=template, data=data)
    artifact = plan.render_templates(
        config=config,
        name=artifact_name,
        description="Generating '{}'".format(artifact_name),
    )
    context.artifacts[artifact_name]["generated"] = True
    return artifact


def read_json_artifact(plan, artifact_name, json_field, file_path="*.json"):
    input_dir = "/_input"
    result = plan.run_sh(
        image="badouralix/curl-jq",
        run="jq -j {field} {input}/{path}".format(
            field=json_field, input=input_dir, path=file_path
        ),
        files={input_dir: artifact_name},
    )
    return result.output


def send_funds(plan, context, to, amount="10ether"):
    http_rpc_url = context.ethereum.all_participants[0].el_context.rpc_http_url
    funded_private_key = context.ethereum.pre_funded_accounts[0].private_key
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
        description="Depositing funds to account",
    )


def generate_env_vars(context, env_vars):
    return {
        env_var_name: expand(context, env_var_value)
        for env_var_name, env_var_value in env_vars.items()
    }


def expand(context, var):
    """
    Replaces values starting with `$` to their dynamically evaluated counterpart.
    Values starting with `$$` are not expanded, and the leading `$` is removed.

    Example: "$service.some_service_name.ip_address" -> <some_service_name's ip address>
    """
    if not var.startswith("$"):
        return var

    if var.startswith("$$"):
        return var[1:]

    path = var[1:].split(".")
    value = context.data
    for field in path:
        value = value.get(field, None)
        if value == None:
            break

    if value == None or type(value) == type({}):
        fail("Invalid path: " + var)

    return value
