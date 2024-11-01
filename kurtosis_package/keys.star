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

    for key in keys:
        name = key["name"]
        key_type = key["type"]
        info = generate_keys(plan, generator_service.name, key_type, name)

        if key_type == "ecdsa":
            shared_utils.send_funds(plan, context, info["address"])

        keys_data[name] = info

    plan.remove_service(generator_service.name)


def generate_keys(plan, egnkey_service_name, key_type, artifact_name):
    output_dir = "/_output"

    cmd = "rm -rf {output} && mkdir -p {output} && \
    egnkey generate --key-type {type} --num-keys 1 --output-dir {output} ; \
    cat {output}/password.txt | tr -d '\n'".format(
        output=output_dir, type=key_type
    )

    result = plan.exec(
        service_name=egnkey_service_name,
        recipe=ExecRecipe(command=["sh", "-c", cmd]),
        description="Generating " + key_type + " key",
    )
    password = result["output"]

    _file_artifact = plan.store_service_files(
        service_name=egnkey_service_name,
        src=output_dir,
        name=artifact_name,
        description="Storing " + key_type + " key",
    )

    cmd = "cat {}/private_key_hex.txt | tr -d '\n'".format(output_dir)

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
        address = shared_utils.read_json_artifact(plan, artifact_name, ".address")
        # Prepend the address with "0x" manually
        keys_info["address"] = "0x" + address

    return keys_info
