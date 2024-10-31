shared_utils = import_module("../shared_utils.star")


def deploy_generic_contract(plan, context, deployment):
    deployment_name = deployment["name"]
    repo = deployment["repo"]
    contracts_path = deployment.get("contracts_path", ".")
    script_path = deployment["script"]
    extra_args = deployment.get("extra_args", "")
    env_vars = shared_utils.generate_env_vars(context, deployment.get("env", {}))
    verify = deployment.get("verify", False)
    input = deployment.get("input", {})

    root = "/app/" + contracts_path + "/"

    input_files = generate_input_files(plan, context, input, root)

    deployer_img = gen_deployer_img(repo, deployment["ref"], contracts_path)

    store_specs, output_renames = generate_store_specs(
        context, root, deployment.get("output", {})
    )

    pre_cmd, input_files = rename_input_files(input_files)
    deploy_cmd = generate_deploy_cmd(context, script_path, extra_args, verify)
    post_cmd = generate_post_cmd(output_renames)

    cmd = generate_cmd([pre_cmd, deploy_cmd, post_cmd])

    # Deploy the Incredible Squaring AVS contracts
    result = plan.run_sh(
        image=deployer_img,
        run=cmd,
        files=input_files,
        store=store_specs,
        env_vars=env_vars,
        description="Deploying '{}'".format(deployment_name),
        wait="600s",
    )
    return result


def gen_deployer_img(repo, ref, path):
    name = repo.rstrip(".git").split("/")[-1]
    ref_name = ref.replace("/", "_")
    # Generate a unique identifier for the image
    uid = hash(str(repo + chr(0) + ref + chr(0) + path)) % 1000000
    return ImageBuildSpec(
        image_name="{name}_{ref}_deployer_{uid}".format(
            name=name, ref=ref_name, uid=uid
        ),
        build_context_dir="../dockerfiles/",
        build_file="contract_deployer.Dockerfile",
        build_args={
            "CONTRACTS_REPO": repo,
            "CONTRACTS_REF": ref,
            "CONTRACTS_PATH": path,
        },
    )


def generate_input_files(plan, context, input, root):
    input_files = shared_utils.generate_input_files(
        plan, context, input, allow_dirs=False
    )
    return {expand_path(context, root, path): art for path, art in input_files.items()}


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


def rename_input_files(input_files):
    renamed_input_files = {}
    cmds = []
    for dst_path, artifact_name in input_files.items():
        src_path = "/var/__" + artifact_name
        renamed_input_files[src_path] = artifact_name
        # Create the directory if it doesn't exist
        cmd1 = "mkdir -p {}".format(dst_path)
        # Copy the artifact to the directory
        cmd2 = "cp -RT {} {}".format(src_path, dst_path)
        # Remove the temporary directory
        cmd3 = "rm -rf {}".format(src_path)
        cmds.extend([cmd1, cmd2, cmd3])

    return " && ".join(cmds), renamed_input_files


def generate_deploy_cmd(context, script_path, extra_args, verify):
    http_rpc_url = context.ethereum.all_participants[0].el_context.rpc_http_url
    private_key = context.ethereum.pre_funded_accounts[0].private_key
    verify_args = get_verify_args(context) if verify else ""
    # We use 'set -e' to fail the script if any command fails
    cmd = "set -e ; forge script --rpc-url {} --private-key 0x{} {} --broadcast -vvv {} {}".format(
        http_rpc_url, private_key, verify_args, script_path, extra_args
    )
    return cmd


def generate_post_cmd(renames):
    return " && ".join(["mv {} {}".format(src, dst) for src, dst in renames])


def generate_cmd(cmds):
    return " ; ".join([c for c in cmds if c != ""])


def get_verify_args(context):
    verify_url = context.ethereum.blockscout_sc_verif_url
    if verify_url == "":
        return ""
    return "--verify --verifier blockscout --verifier-url {}/api?".format(verify_url)
