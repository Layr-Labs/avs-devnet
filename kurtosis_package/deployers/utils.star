shared_utils = import_module("../shared_utils.star")

# Foundry image (arm64-compatible)
FOUNDRY_IMAGE = "ghcr.io/foundry-rs/foundry:latest"


def deploy_generic_contract(plan, context, deployment):
    name = deployment["name"]
    repo = deployment["repo"]
    is_remote_repo = repo.startswith("https://") or repo.startswith("http://")
    contracts_path = deployment.get("contracts_path", ".")
    script = deployment["script"]
    extra_args = deployment.get("extra_args", "")
    env = deployment.get("env", {})
    verify = deployment.get("verify", False)
    input = deployment.get("input", {})
    output = deployment.get("output", {})
    addresses = deployment.get("addresses", {})

    # Prefix for the artifact generated during variable expansion
    prefix = "contract_{}_expanded_env_var_".format(name)
    env_vars = shared_utils.generate_env_vars(plan, context, env, prefix)

    contract_name = None

    root = "/app/" + contracts_path + "/"

    input_artifacts = generate_input_artifacts(plan, context, input, root)

    if is_remote_repo:
        deployer_img = gen_deployer_img(repo, deployment["ref"], contracts_path)
    else:
        deployer_img = FOUNDRY_IMAGE
        split_path = script.split(".sol:")
        script_path = script
        # In case the contract name is not provided, we assume the contract name is in the script name
        # Examples: "MyContract.sol" -> "MyContract", "MyContract.sol:MyContract2" -> "MyContract2"
        if len(split_path) == 1:
            contract_name = script.split("/")[-1].rstrip(".sol").rstrip(".s")
        else:
            contract_name = split_path[1]
            script_path = split_path[0] + ".sol"

        input_artifacts.append(("/app/", name + "-script"))

    store_specs, output_renames = generate_store_specs(root, output)

    pre_cmd, input_files = rename_input_files(input_artifacts)
    move_to_dir_cmd = "cd " + root
    deploy_cmd = generate_deploy_cmd(context, script, contract_name, extra_args, verify)
    post_cmd = generate_post_cmd(output_renames)

    cmd = generate_cmd([pre_cmd, move_to_dir_cmd, deploy_cmd, post_cmd])

    result = plan.run_sh(
        image=deployer_img,
        run=cmd,
        files=input_files,
        store=store_specs,
        env_vars=env_vars,
        description="Deploying '{}'".format(name),
        wait=None,
    )
    context.data["addresses"][name] = extract_addresses(plan, context, addresses)
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


def generate_input_artifacts(plan, context, input, root):
    artifacts, files = parse_input_files(input, root)
    shared_utils.ensure_all_generated(plan, context, artifacts)
    return files


def parse_input_files(input_args, root_dir):
    artifacts = []
    files = []
    for path, artifact_names in input_args.items():
        if type(artifact_names) == type(""):
            artifact_names = [artifact_names]

        expanded_path = expand_path(root_dir, path)
        artifacts.extend(artifact_names)
        files.extend([(expanded_path, art) for art in artifact_names])

    return artifacts, files


def generate_store_specs(root_dir, output_args):
    output = []
    renames = []

    for artifact_name, output_info in output_args.items():
        if type(output_info) == type(""):
            output_info = struct(path=output_info, rename=None)
        else:
            rename = output_info.get("rename", None)
            output_info = struct(path=output_info["path"], rename=rename)

        expanded_path = expand_path(root_dir, output_info.path)
        if output_info.rename != None:
            path_stem = expanded_path.rsplit("/", 1)[0]
            renamed_file = path_stem + "/" + output_info.rename
            renames.append((expanded_path, renamed_file))
            expanded_path = renamed_file

        output.append(StoreSpec(src=expanded_path, name=artifact_name))

    return output, renames


def expand_path(root_dir, path):
    return (root_dir + path).replace("//", "/")


def rename_input_files(input_files):
    renamed_input_files = {}
    cmds = []
    for dst_path, artifact_name in input_files:
        src_path = "/var/__" + artifact_name
        renamed_input_files[src_path] = artifact_name
        # Create the directory if it doesn't exist
        cmd1 = "mkdir -p {}".format(dst_path)
        # Copy the artifact to the directory
        cmd2 = "cp -RT {} {}".format(src_path, dst_path)
        cmds.extend([cmd1, cmd2])

    return " && ".join(cmds), renamed_input_files


def generate_deploy_cmd(context, script, contract_name, user_extra_args, verify):
    http_rpc_url = context.ethereum.all_participants[0].el_context.rpc_http_url
    private_key = context.ethereum.pre_funded_accounts[0].private_key
    verify_args = get_verify_args(context) if verify else ""
    target_contract_arg = ("--tc " + contract_name) if contract_name != None else ""
    extra_args = " ".join([verify_args, target_contract_arg])

    cmd = "forge script --rpc-url {} --private-key {} {} --broadcast --non-interactive -vvv {} {}".format(
        http_rpc_url, private_key, extra_args, script, user_extra_args
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


def extract_addresses(plan, context, addresses):
    extracted_addresses = {}
    for name, locator in addresses.items():
        split_locator = locator.split(":")

        if len(split_locator) != 2:
            fail("Locator '{}' must have exactly one ':' character".format(locator))

        artifact, path = split_locator
        address = shared_utils.read_json_artifact(plan, artifact, path)
        extracted_addresses[name] = address

    return extracted_addresses
