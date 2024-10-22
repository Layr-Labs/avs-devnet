shared_utils = import_module("./shared_utils.star")

def generate_all_keystores(plan, context, keystores):
    if len(keystores) == 0:
        return
    keystore_data = context.data.get("keystores", {})

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

    for keystore in keystores:
        name = keystore["name"]
        key_type = keystore["type"]
        info = generate_keystore(plan, generator_service.name, key_type, name)

        if key_type == "ecdsa":
            shared_utils.send_funds(plan, context, info["address"])

        keystore_data[name] = info

    plan.remove_service(generator_service.name)


def generate_keystore(plan, egnkey_service_name, key_type, artifact_name):
    tmp_dir = "/_tmp"
    output_dir = "/_output"

    cmd = "rm -rf {tmp} && mkdir -p {output} && egnkey generate --key-type {type} --num-keys 1 --output-dir {tmp} && mv {tmp}/keys/1.{type}.key.json {output} ; cat {tmp}/password.txt | tr -d '\n'".format(
        tmp=tmp_dir, output=output_dir, type=key_type
    )

    result = plan.exec(
        service_name=egnkey_service_name,
        recipe=ExecRecipe(command=["sh", "-c", cmd]),
        description="Generating " + key_type + " key",
    )
    password = result["output"]

    _file_artifact = plan.store_service_files(
        service_name=egnkey_service_name,
        src=output_dir + "/1." + key_type + ".key.json",
        name=artifact_name,
        description="Storing " + key_type + " key",
    )
    keystore_info = {
        "name": artifact_name,
        "type": key_type,
        "password": password,
    }

    if key_type == "ecdsa":
        address = shared_utils.read_json_artifact(plan, artifact_name, ".address")
        keystore_info["address"] = address

    return keystore_info
