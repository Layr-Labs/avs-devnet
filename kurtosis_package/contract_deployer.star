shared_utils = import_module("./shared_utils.star")


def deploy(plan, context, deployment):
    deployment_type = deployment.get("type", "")

    if deployment_type.lower() == "eigenlayer":
        deploy_eigenlayer(plan, context, deployment)
    else:
        deploy_generic_contract(plan, context, deployment)


EL_DEFAULT = {
    "name": "EigenLayer",
    "repo": "https://github.com/Layr-Labs/eigenlayer-contracts.git",
    "ref": "dev",
    "contracts_path": ".",
    "script": "script/deploy/devnet/M2_Deploy_From_Scratch.s.sol:Deployer_M2",
    "extra_args": "--sig 'run(string memory configFileName)' -- deploy_from_scratch.config.json",
    "input": {"script/configs/devnet/": "eigenlayer_deployment_input"},
    "output": {
        "eigenlayer_addresses": {
            "path": "script/output/devnet/M2_from_scratch_deployment_data.json",
            "rename": "eigenlayer_deployment_output.json",
        }
    },
}

EL_CONFIG_TEMPLATE = """
{
  "maintainer": "example@example.org",
  "multisig_addresses": {
    "operationsMultisig": "{{.deployer_address}}",
    "pauserMultisig": "{{.deployer_address}}",
    "executorMultisig": "{{.deployer_address}}"
  },
  "strategies": [],
  "strategyManager": {
    "init_paused_status": 0,
    "init_withdrawal_delay_blocks": 1
  },
  "eigenPod": {
    "PARTIAL_WITHDRAWAL_FRAUD_PROOF_PERIOD_BLOCKS": 1,
    "MAX_RESTAKED_BALANCE_GWEI_PER_VALIDATOR": "32000000000"
  },
  "eigenPodManager": {
    "init_paused_status": 30
  },
  "delayedWithdrawalRouter": {
    "init_paused_status": 0,
    "init_withdrawal_delay_blocks": 1
  },
  "slasher": {
    "init_paused_status": 0
  },
  "delegation": {
    "init_paused_status": 0,
    "init_withdrawal_delay_blocks": 1
  },
  "rewardsCoordinator": {
    "init_paused_status": 0,
    "CALCULATION_INTERVAL_SECONDS": 604800,
    "MAX_REWARDS_DURATION": 6048000,
    "MAX_RETROACTIVE_LENGTH": 7776000,
    "MAX_FUTURE_LENGTH": 2592000,
    "GENESIS_REWARDS_TIMESTAMP": 1710979200,
    "rewards_updater_address": "{{.deployer_address}}",
    "activation_delay": 7200,
    "calculation_interval_seconds": 604800,
    "global_operator_commission_bips": 1000
  },
  "ethPOSDepositAddress": "0x00000000219ab540356cBB839Cbe05303d7705Fa"
}
"""


def deploy_eigenlayer(plan, context, deployment):
    plan.print("Initiating EigenLayer deployment")

    generate_el_config(plan, context, deployment)
    el_args = dict(EL_DEFAULT)
    el_args.update(deployment)
    deploy_generic_contract(plan, context, el_args)


def generate_el_config(plan, context, deployment):
    # TODO: generate it manually to allow for more customization
    context.artifacts["eigenlayer_deployment_input"] = {
        "files": {"deploy_from_scratch.config.json": EL_CONFIG_TEMPLATE}
    }


def deploy_generic_contract(plan, context, deployment):
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
