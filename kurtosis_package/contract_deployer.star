def deploy(plan, context, deployment):
    repo = deployment["repo"]
    ref = deployment["ref"]
    contracts_path = deployment["contracts_path"]
    deployer_img = gen_deployer_img(repo, ref, contracts_path)

    chain_id = context.ethereum.network_id
    output_dir = "/app/{}/script/output/{chain_id}/".format(contracts_path, chain_id=chain_id)

    http_rpc_url = context.ethereum.all_participants[0].el_context.rpc_http_url
    private_key = context.ethereum.pre_funded_accounts[0].private_key

    # Deploy the Incredible Squaring AVS contracts
    result = plan.run_sh(
        image=deployer_img,
        run="forge script ./script/IncredibleSquaringDeployer.s.sol --rpc-url ${HTTP_RPC_URL}  --private-key 0x${PRIVATE_KEY} --broadcast -vvv",
        env_vars={
            "HTTP_RPC_URL": http_rpc_url,
            "PRIVATE_KEY": private_key,
        },
        files={output_dir: "eigenlayer_addresses"},
        store=[
            StoreSpec(
                src=output_dir + "credible_squaring_avs_deployment_output.json",
                name="avs_addresses",
            )
        ],
        description="Deploying Incredible Squaring contracts",
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
