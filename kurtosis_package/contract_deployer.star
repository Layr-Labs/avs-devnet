shared_utils = import_module("./shared_utils.star")
eigenlayer = import_module("./deployers/eigenlayer.star")
utils = import_module("./deployers/utils.star")


def deploy(plan, context, deployment):
    deployment_type = deployment.get("type", "default").lower()

    if deployment_type == "eigenlayer":
        eigenlayer.deploy(plan, context, deployment)
    elif deployment_type == "default":
        utils.deploy_generic_contract(plan, context, deployment)
    else:
        fail("Unknown deployment type: {}".format(deployment_type))
