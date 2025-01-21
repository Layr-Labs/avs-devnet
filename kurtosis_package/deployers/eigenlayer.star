shared_utils = import_module("../shared_utils.star")
utils = import_module("./utils.star")


def deploy(plan, context, deployment):
    el_args = get_version_args(deployment) | deployment
    el_name = el_args["name"]
    plan.print("Initiating {} deployment".format(el_name))

    context.data["addresses"][el_name] = {}

    strategies = parse_strategies(el_args["strategies"])

    token_address = None
    if len(strategies) > 0:
        token_address = deploy_mocktoken(plan, context, el_name, el_args["verify"])

    config_name = generate_el_config(plan, context, token_address, strategies)

    el_args["input"] = dict(el_args["input"])

    # Allow any number of replacements, so users can change the config path
    # when using an unsupported version
    for key, value in el_args["input"].items():
        if value == CONFIG_ARTIFACT_PLACEHOLDER:
            el_args["input"][key] = config_name

    el_args["addresses"] = el_args["addresses"] | generate_addresses_arg(
        "eigenlayer_addresses", strategies
    )

    utils.deploy_generic_contract(plan, context, el_args)

    context.data["addresses"][el_name]["mocktoken"] = token_address

    whitelist_strategies(plan, context, el_name, strategies)
    register_operators(plan, context, el_name, el_args["operators"])


def parse_strategies(raw_strategies):
    parsed_strategies = []
    for strategy in raw_strategies:
        parsed = dict(DEFAULT_STRATEGY)
        if type(strategy) == type(""):
            parsed["name"] = strategy
        else:
            parsed.update(strategy)
        parsed_strategies.append(parsed)
    return parsed_strategies


def generate_addresses_arg(el_output, strategies):
    addresses = {}
    for name in EL_CONTRACT_NAMES:
        addresses[name] = el_output + ":.addresses." + name

    # Specify addresses to read from the output
    for strategy in strategies:
        name = strategy["name"]
        path = el_output + ":.addresses.strategies." + name
        addresses[name] = path

    return addresses


def get_version_args(deployment):
    ref = deployment.get("ref", "dev")
    deployment_version = deployment.get("version", ref)
    if deployment_version == "":
        return EL_DEPLOY_ARGS_LATEST

    version_unprefixed = deployment_version.lstrip("v")

    if version_unprefixed[0].isdigit():
        version = version_unprefixed.split("-")[0].split(".")
        major = int(version[0])
        minor = int(version[1])
        patch = int(version[2])
        # v0.4.2-mainnet-pepe and below
        if major == 0 and (minor < 4 or (minor == 4 and patch <= 2)):
            return EL_DEPLOY_ARGS_v0_4_2

        # Other versions before v1.0.0
        if major == 0:
            return EL_DEPLOY_ARGS_v0_5_3

    return EL_DEPLOY_ARGS_LATEST


def deploy_mocktoken(plan, context, deployment_name, verify):
    repo = "https://github.com/Layr-Labs/incredible-squaring-avs.git"
    ref = "83e64c8f11439028186380ef0ed35eea6316ec47"
    path = "contracts"

    deployer_img = utils.gen_deployer_img(repo, ref, path)

    http_rpc_url = context.ethereum.all_participants[0].el_context.rpc_http_url
    private_key = context.ethereum.pre_funded_accounts[0].private_key

    verify_args = utils.get_verify_args(context) if verify else ""

    cmd = "set -e ; forge create --broadcast --rpc-url {} --private-key {} {} src/ERC20Mock.sol:ERC20Mock 2> /dev/null \
    | awk '/Deployed to: .*/{{ print $3 }}' | tr -d '\"\n'".format(
        http_rpc_url,
        private_key,
        verify_args,
    )
    result = plan.run_sh(
        image=deployer_img,
        run=cmd,
        description="Deploying 'ERC20Mock'",
    )
    token_address = result.output
    # run_sh doesn't check the exit code of the command, so we verify the result.
    plan.verify(
        value=token_address,
        assertion="!=",
        target_value="",
        description="Verifying token deployment",
    )

    return token_address


def generate_el_config(plan, context, token_address, strategies):
    formatted_strategies = format_strategies(context, token_address, strategies)
    data = {
        "deployer_address": context.data["deployer_address"],
        "strategies": formatted_strategies,
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


def format_strategies(context, token_address, strategies):
    formatted_strategies = []
    for strategy in strategies:
        formatted_strategies.append(
            json.encode(
                {
                    "max_deposits": strategy["max_deposits"],
                    "max_per_deposit": strategy["max_per_deposit"],
                    "token_address": token_address,
                    # This is the associated name in the output file
                    "token_symbol": strategy["name"],
                }
            )
        )
    return ", ".join(formatted_strategies)


def whitelist_strategies(plan, context, deployment_name, strategies):
    if len(strategies) == 0:
        return

    data = context.data
    addresses = data["addresses"][deployment_name]
    strategy_params = ",".join([addresses[strategy["name"]] for strategy in strategies])
    flag_params = ",".join(["true" for _ in strategies])
    cmd = "set -e ; cast send --rpc-url {rpc} --private-key {pk} \
    {addr} 'addStrategiesToDepositWhitelist(address[],bool[])' '[{strategy_params}]' '[{flag_params}]'".format(
        rpc=data["http_rpc_url"],
        pk=data["deployer_private_key"],
        addr=addresses["strategyManager"],
        strategy_params=strategy_params,
        flag_params=flag_params,
    )
    plan.run_sh(
        image=utils.FOUNDRY_IMAGE, run=cmd, description="Whitelisting strategies"
    )


def register_operators(plan, context, deployment_name, operators):
    data = context.data
    for operator in operators:
        operator_name = operator["name"]
        keys_name = operator["keys"]
        strategies = operator.get("strategies", [])

        operator_keys = data["keys"][keys_name]
        addresses = data["addresses"][deployment_name]

        send_cmd = (
            "cast send --confirmations 0 --rpc-url {rpc} --private-key {pk}".format(
                rpc=data["http_rpc_url"], pk=operator_keys["private_key"]
            )
        )
        cmds = ["set -e"]
        cmds.append(
            # NOTE: we don't use the zero address for backwards compatibility
            "{} {addr} 'registerAsOperator((address,address,uint32),string)' \"(0x000000000000000000000000000000000000de4d,0x0000000000000000000000000000000000000000,0)\" {metadata}".format(
                send_cmd,
                addr=addresses["delegation"],
                metadata=DEFAULT_METADATA_URI,
            )
        )

        manager_addr = addresses["strategyManager"]
        # TODO: allow other tokens
        token = addresses["mocktoken"]
        for strategy, amount in strategies.items():
            strategy_addr = addresses[strategy]
            # Mint tokens
            cmds.append(
                "{} {token} 'mint(address,uint256)' {addr} {amount}".format(
                    send_cmd,
                    token=token,
                    addr=operator_keys["address"],
                    amount=amount,
                )
            )
            # Approve token transfer
            cmds.append(
                "{} {addr} 'approve(address,uint256)(bool)' {strategy} {amount}".format(
                    send_cmd, addr=token, strategy=manager_addr, amount=amount
                )
            )
            # Perform deposit
            cmds.append(
                "{} {addr} 'depositIntoStrategy(address,address,uint256)(uint256)' {strategy} {token} {amount}".format(
                    send_cmd,
                    addr=manager_addr,
                    strategy=strategy_addr,
                    token=token,
                    amount=amount,
                )
            )

        plan.run_sh(
            image=utils.FOUNDRY_IMAGE,
            run=" ; ".join(cmds),
            description="Registering operator '{}'".format(operator_name),
        )


# CONSTANTS

MAX_UINT256 = 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF
DEFAULT_METADATA_URI = "https://raw.githubusercontent.com/Layr-Labs/eigenlayer-cli/79add3518f856c71faa3b95b383e35df370bcc52/pkg/operator/config/metadata-example.json"

DEFAULT_STRATEGY = {
    "max_deposits": MAX_UINT256,
    "max_per_deposit": MAX_UINT256,
}

EL_CONTRACT_NAMES = [
    "avsDirectory",
    "avsDirectoryImplementation",
    "baseStrategyImplementation",
    "delegation",
    "delegationImplementation",
    "eigenLayerPauserReg",
    "eigenLayerProxyAdmin",
    "eigenPodBeacon",
    "eigenPodImplementation",
    "eigenPodManager",
    "eigenPodManagerImplementation",
    "emptyContract",
    "rewardsCoordinator",
    "rewardsCoordinatorImplementation",
    "slasher",
    "slasherImplementation",
    "strategyManager",
    "strategyManagerImplementation",
]

EL_DEFAULT_ARGS = {
    "name": "EigenLayer",
    "repo": "https://github.com/Layr-Labs/eigenlayer-contracts.git",
    "extra_args": "--sig 'run(string memory configFileName)' -- deploy_from_scratch.config.json",
    "verify": False,
    "contracts_path": ".",
    "addresses": {},
    "strategies": [],
    "operators": [],
}

# Placeholder for the dynamically generated artifact name
CONFIG_ARTIFACT_PLACEHOLDER = "+$+CONFIG_ARTIFACT+$+"

EL_DEPLOY_ARGS_v0_4_2 = {
    "ref": "v0.4.2-mainnet-pepe",
    "script": "script/deploy/devnet/M2_Deploy_From_Scratch.s.sol:Deployer_M2",
    "input": {"script/configs/devnet/": CONFIG_ARTIFACT_PLACEHOLDER},
    "output": {
        "eigenlayer_addresses": {
            "path": "script/output/devnet/M2_from_scratch_deployment_data.json",
            "rename": "eigenlayer_deployment_output.json",
        }
    },
} | EL_DEFAULT_ARGS

EL_DEPLOY_ARGS_v0_5_3 = {
    "ref": "v0.5.3",
    "script": "script/deploy/local/Deploy_From_Scratch.s.sol:DeployFromScratch",
    "input": {"script/configs/": CONFIG_ARTIFACT_PLACEHOLDER},
    "output": {
        "eigenlayer_addresses": {
            "path": "script/output/devnet/local_from_scratch_deployment_data.json",
            "rename": "eigenlayer_deployment_output.json",
        }
    },
} | EL_DEFAULT_ARGS

EL_DEPLOY_ARGS_LATEST = {
    "ref": "dev",
    "script": "script/deploy/local/Deploy_From_Scratch.s.sol:DeployFromScratch",
    "input": {"script/configs/": CONFIG_ARTIFACT_PLACEHOLDER},
    "output": {
        "eigenlayer_addresses": {
            "path": "script/output/devnet/M2_from_scratch_deployment_data.json",
            "rename": "eigenlayer_deployment_output.json",
        }
    },
} | EL_DEFAULT_ARGS

EL_CONFIG_TEMPLATE = """
{
  "maintainer": "example@example.org",
  "multisig_addresses": {
    "operationsMultisig": "{{.deployer_address}}",
    "communityMultisig": "{{.deployer_address}}",
    "pauserMultisig": "{{.deployer_address}}",
    "executorMultisig": "{{.deployer_address}}",
    "timelock": "{{.deployer_address}}"
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
  "allocationManager": {
    "init_paused_status": 0,
    "DEALLOCATION_DELAY": 900,
    "ALLOCATION_CONFIGURATION_DELAY": 1200
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
    "global_operator_commission_bips": 1000,
    "default_operator_split_bips": 1000,
    "OPERATOR_SET_GENESIS_REWARDS_TIMESTAMP": 1720656000,
    "OPERATOR_SET_MAX_RETROACTIVE_LENGTH": 2592000
  },
  "ethPOSDepositAddress": "0x4242424242424242424242424242424242424242"
}
"""
