# NOTE: this is a temporary workaround due to foundry-rs not having arm64 images
FOUNDRY_IMAGE = ImageBuildSpec(
    image_name="Layr-Labs/foundry",
    build_context_dir="./dockerfiles/",
    build_file="contract_deployer.Dockerfile",
    target_stage="foundry",
)


def ensure_all_generated(plan, context, artifacts):
    """
    Ensures all the given artifacts are generated.
    """
    for artifact_name in artifacts:
        ensure_generated(plan, context, artifact_name)


def ensure_generated(plan, context, artifact_name):
    """
    Ensures an artifact was already generated, and returns its artifact name.
    """
    # If it's not in the context, we assume it's auto-generated
    if artifact_name not in context.artifacts:
        return artifact_name

    # If it was already generated, we skip it
    if context.artifacts[artifact_name].get("generated", False):
        return artifact_name

    artifact_files = context.artifacts[artifact_name].get("files", {})
    additional_data = context.artifacts[artifact_name].get("additional_data", {})

    # Make a copy of the context data to allow for modifications
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
        run="jq -j '{field}' {input}/{path}".format(
            field=json_field, input=input_dir, path=file_path
        ),
        files={input_dir: artifact_name},
    )
    return result.output


def send_funds(plan, context, to, amount="10ether"):
    http_rpc_url = context.ethereum.all_participants[0].el_context.rpc_http_url
    funded_private_key = context.ethereum.pre_funded_accounts[0].private_key
    cmd = "cast send --value {} --private-key {} --rpc-url {} {}".format(
        amount, funded_private_key, http_rpc_url, to
    )
    plan.run_sh(
        image=FOUNDRY_IMAGE,
        run=cmd,
        description="Depositing funds to account",
    )


def generate_env_vars(plan, context, env_vars):
    return {
        env_var_name: expand(plan, context, env_var_value)
        for env_var_name, env_var_value in env_vars.items()
    }


def expand(plan, context, var):
    """
    Replaces templates containing double brackets ("{{") to their dynamically evaluated counterpart.

    Example: "{{.service.some_service_name.ip_address}}" -> <some_service_name's ip address>
    """
    # NOTE: this is just an optimization to avoid template rendering if it doesn't need it
    if var.find("{{") == -1:
        return var

    file_name = "expanded.txt"

    artifact = plan.render_templates(
        config={file_name: struct(template=var, data=context.data)},
        description="Expanding envvar '{}'".format(var),
    )
    result = plan.run_sh(
        run="cat /artifact/" + file_name,
        files={"/artifact": artifact},
    )
    expanded_value = result.output
    return expanded_value
