shared_utils = import_module("./shared_utils.star")


def generate_all_keys(plan, context, keys):
    if len(keys) == 0:
        return
    keys_data = context.data["keys"]

    generator_service = plan.add_service(
        "egnkey-service",
        config=ServiceConfig(
            image=ImageBuildSpec(
                image_name="egnkey",
                build_context_dir="./dockerfiles/",
                build_file="egnkey.Dockerfile",
            ),
            entrypoint=["sleep", "99999"],
            description="Spinning up EigenLayer key generator service",
        ),
    )
    generator = generator_service.name

    for i, key in enumerate(keys):
        info = parse_key_info(plan, context, generator, key, i)
        name = info["name"]

        if "address" in info:
            shared_utils.send_funds(plan, context, info["address"])

        keys_data[name] = info

    plan.remove_service(generator)


def parse_key_info(plan, context, generator, key, i):
    info = {}
    name = key.get("name", "key{}".format(i))
    key_type = key.get("type", "ecdsa")

    address = key.get("address")
    private_key = key.get("private_key")

    should_be_generated = not (address or private_key)
    info["name"] = name
    info["type"] = key_type

    if address:
        if key_type != "ecdsa":
            fail("Only ECDSA keys can have an address")
        info["address"] = address

    if private_key:
        # TODO: derive address from private key
        info["private_key"] = private_key

    if should_be_generated:
        info.update(generate_keys(plan, generator, key_type, name))

    return info


def generate_keys(plan, egnkey_service_name, key_type, artifact_name):
    output_dir = "/_output"

    cmd = "set -e ; rm -rf {output} && \
    egnkey generate --key-type {type} --num-keys 1 --output-dir {output}".format(
        output=output_dir, type=key_type
    )

    result = plan.exec(
        service_name=egnkey_service_name,
        recipe=ExecRecipe(command=["sh", "-c", cmd]),
        description="Generating " + key_type + " key",
    )

    artifact_name = plan.store_service_files(
        service_name=egnkey_service_name,
        src=output_dir,
        name=artifact_name,
        description="Storing " + key_type + " key",
    )

    # NOTE: we do this in another step to avoid egnkey's output from being stored
    cmd = "set -e ; cat {}/password.txt | tr -d '\n'".format(output_dir)

    result = plan.exec(
        service_name=egnkey_service_name,
        recipe=ExecRecipe(command=["sh", "-c", cmd]),
        description="Extracting keystore password",
    )
    password = result["output"]

    cmd = "set -e ; cat {}/private_key_hex.txt | tr -d '\n'".format(output_dir)

    result = plan.exec(
        service_name=egnkey_service_name,
        recipe=ExecRecipe(command=["sh", "-c", cmd]),
        description="Extracting private key",
    )
    # NOTE: this is in hexa for ECDSA and decimal for BLS
    private_key = result["output"]

    keys_info = {
        "name": artifact_name,
        "type": key_type,
        "password": password,
        "private_key": private_key,
    }

    if key_type == "ecdsa":
        address = shared_utils.read_json_artifact(
            plan, artifact_name, ".address", file_path="keys/1.ecdsa.key.json"
        )
        # Prepend the address with "0x" manually
        keys_info["address"] = "0x" + address

    return keys_info
