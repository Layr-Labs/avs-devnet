def identity(x):
    return x


def generate_input_files(plan, context, input_args, mapper=identity, allow_dirs=True):
    files = {}

    for path, artifact_names in input_args.items():
        if type(artifact_names) == type(""):
            artifact_names = [artifact_names]
        if len(artifact_names) == 0:
            continue
        if len(artifact_names) == 1:
            artifact = generate_artifact(plan, context, artifact_names[0])
        elif allow_dirs:
            artifacts = [generate_artifact(plan, context, n) for n in artifact_names]
            artifact = Directory(artifact_names=artifacts)
        else:
            fail(
                "Only single artifacts allowed: '{}' (specified: {})".format(
                    path, artifact_names
                )
            )

        files[mapper(path)] = artifact

    return files


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


def read_json_artifact(plan, artifact_name, json_field):
    input_dir = "/_input"
    result = plan.run_sh(
        image="badouralix/curl-jq",
        run="jq -j {field} {input}/*.json".format(field=json_field, input=input_dir),
        files={input_dir: artifact_name},
        wait="1s",
    )
    return result.output
