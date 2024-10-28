utils = import_module("./utils.star")


def deploy(plan, context, deployment):
    plan.print("Initiating EigenLayer deployment")
    strategies = deployment.get("strategies", [])
    if len(strategies) > 0:
        deploy_mocktoken(plan, context)

    config_name = generate_el_config(plan, context, deployment)
    el_args = EL_DEFAULT | deployment
    # TODO: insert as list if user specifies same path
    el_args["input"] = el_args["input"] | {"script/configs/devnet/": config_name}
    utils.deploy_generic_contract(plan, context, el_args)


def deploy_mocktoken(plan, context):
    repo = "https://github.com/Layr-Labs/incredible-squaring-avs.git"
    ref = "83e64c8f11439028186380ef0ed35eea6316ec47"
    path = "contracts"
    deployer_img = utils.gen_deployer_img(repo, ref, path)
    http_rpc_url = context.ethereum.all_participants[0].el_context.rpc_http_url
    private_key = context.ethereum.pre_funded_accounts[0].private_key
    cmd = "set -e ; forge create --rpc-url {} --private-key 0x{} src/ERC20Mock.sol:ERC20Mock \
    | awk '/Deployed to: .*/{{ print $3 }}' | tr -d '\"\n'".format(
        http_rpc_url,
        private_key,
    )
    result = plan.run_sh(
        image=deployer_img,
        run=cmd,
        description="Deploying 'ERC20Mock'",
    )
    token_address = result.output
    context.data["addresses"]["mocktoken"] = token_address

    return token_address


def generate_el_config(plan, context, deployment):
    strategies = format_strategies(context, deployment.get("strategies", []))
    data = {
        "deployer_address": context.data["deployer_address"],
        "strategies": strategies,
    }
    artifact_name = plan.render_templates(
        config={
            "deploy_from_scratch.config.json": struct(
                template=EL_CONFIG_TEMPLATE,
                data=data,
            )
        },
        description="Generating EigenLayer deployment config",
    )
    return artifact_name


MAX_UINT256 = 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF


def format_strategies(context, strategies):
    formatted_strategies = []
    for strategy in strategies:
        if type(strategy) == type(""):
            strategy = {"name": strategy}

        formatted_strategies.append(
            json.encode(
                {
                    "max_deposits": strategy.get("max_deposits", MAX_UINT256),
                    "max_per_deposit": strategy.get("max_per_deposit", MAX_UINT256),
                    "token_address": context.data["addresses"]["mocktoken"],
                    # This is the associated name in the output file
                    "token_symbol": strategy["name"],
                }
            )
        )
    return ", ".join(formatted_strategies)


# CONSTANTS

EL_DEFAULT = {
    "name": "EigenLayer",
    "repo": "https://github.com/Layr-Labs/eigenlayer-contracts.git",
    "ref": "dev",
    "contracts_path": ".",
    "script": "script/deploy/devnet/M2_Deploy_From_Scratch.s.sol:Deployer_M2",
    "extra_args": "--sig 'run(string memory configFileName)' -- deploy_from_scratch.config.json",
    "input": {},
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
  "strategies": [{{.strategies}}],
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
