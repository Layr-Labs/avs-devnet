def deploy(plan, context, deployment):
    deployment_name = deployment["name"]
    repo = deployment["repo"]
    ref = deployment["ref"]
    contracts_path = deployment["contracts_path"]
    script_path = deployment["script"]

    root = "/app/" + contracts_path + "/"

    input_files = generate_input_files(plan, context, root, deployment.get("input", {}))
    store_specs = generate_store_specs(context, root, deployment.get("output", {}))
    deployer_img = gen_deployer_img(repo, ref, contracts_path)

    http_rpc_url = context.ethereum.all_participants[0].el_context.rpc_http_url
    private_key = context.ethereum.pre_funded_accounts[0].private_key

    # Deploy the Incredible Squaring AVS contracts
    result = plan.run_sh(
        image=deployer_img,
        run="forge script --rpc-url ${HTTP_RPC_URL} --private-key 0x${PRIVATE_KEY} --broadcast -vvv " + script_path,
        env_vars={
            "HTTP_RPC_URL": http_rpc_url,
            "PRIVATE_KEY": private_key,
        },
        files=input_files,
        store=store_specs,
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


# TODO: merge with `generate_service_files`
def generate_input_files(plan, context, root_dir, input_args):
    files = {}

    for path, artifact_name in input_args.items():
        expanded_path = expand_path(context, root_dir, path)
        files[expanded_path] = artifact_name

    return files


def generate_store_specs(context, root_dir, output_args):
    output = []

    for artifact_name, path in output_args.items():
        expanded_path = expand_path(context, root_dir, path)
        output.append(StoreSpec(src=expanded_path, name=artifact_name))

    return output


def expand_path(context, root_dir, path):
    chain_id = context.ethereum.network_id
    return (root_dir + path.format(chain_id=chain_id)).replace("//", "/")
