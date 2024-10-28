shared_utils = import_module("./shared_utils.star")


def deploy(plan, context, deployment):
    deployment_name = deployment["name"]
    repo = deployment["repo"]
    ref = deployment["ref"]
    contracts_path = deployment.get("contracts_path", ".")
    script_path = deployment["script"]
    extra_args = deployment.get("extra_args", "")
    env_vars = shared_utils.generate_env_vars(context, deployment.get("env", {}))

    root = "/app/" + contracts_path + "/"

    def file_mapper(path):
        return expand_path(context, root, path)

    input_files = shared_utils.generate_input_files(
        plan,
        context,
        deployment.get("input", {}),
        mapper=file_mapper,
        allow_dirs=False,
    )
    store_specs, renames = generate_store_specs(
        context, root, deployment.get("output", {})
    )
    deployer_img = gen_deployer_img(repo, ref, contracts_path)

    cmd = generate_cmd(context, script_path, extra_args, renames)

    # Deploy the Incredible Squaring AVS contracts
    result = plan.run_sh(
        image=deployer_img,
        run=cmd,
        files=input_files,
        store=store_specs,
        env_vars=env_vars,
        description="Deploying '{}'".format(deployment_name),
    )


def gen_deployer_img(repo, ref, path):
    name = repo.rstrip(".git").split("/")[-1]
    ref_name = ref.replace("/", "_")
    # Generate a unique identifier for the image
    uid = hash(str(repo + chr(0) + ref + chr(0) + path)) % 1000000
    return ImageBuildSpec(
        image_name="{name}_{ref}_deployer_{uid}".format(
            name=name, ref=ref_name, uid=uid
        ),
        build_context_dir="./dockerfiles/",
        build_file="contract_deployer.Dockerfile",
        build_args={
            "CONTRACTS_REPO": repo,
            "CONTRACTS_REF": ref,
            "CONTRACTS_PATH": path,
        },
    )


def generate_store_specs(context, root_dir, output_args):
    output = []
    renames = []

    for artifact_name, output_info in output_args.items():
        if type(output_info) == type(""):
            output_info = struct(path=output_info, rename=None)
        else:
            output_info = struct(
                path=output_info["path"], rename=output_info.get("rename", None)
            )

        expanded_path = expand_path(context, root_dir, output_info.path)
        if output_info.rename != None:
            path_stem = expanded_path.rsplit("/", 1)[0]
            renamed_file = path_stem + "/" + output_info.rename
            renames.append((expanded_path, renamed_file))
            expanded_path = renamed_file

        output.append(StoreSpec(src=expanded_path, name=artifact_name))

    return output, renames


def expand_path(context, root_dir, path):
    return (root_dir + path).replace("//", "/")


def generate_cmd(context, script_path, extra_args, renames):
    http_rpc_url = context.ethereum.all_participants[0].el_context.rpc_http_url
    private_key = context.ethereum.pre_funded_accounts[0].private_key
    # We use 'set -e' to fail the script if any command fails
    cmd = "set -e ; forge script --rpc-url {} --private-key 0x{} --broadcast -vvv {} {}".format(
        http_rpc_url, private_key, script_path, extra_args
    )
    rename_cmds = ["mv {} {}".format(src, dst) for src, dst in renames]
    if len(rename_cmds) == 0:
        return cmd
    return cmd + " ; " + " && ".join(rename_cmds)
