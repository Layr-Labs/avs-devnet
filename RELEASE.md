# Release Process

<!-- Taken from https://github.com/Layr-Labs/eigenlayer-cli/blob/4ab21e57e58a7bd4008465818ed43910a2507b8b/README.md?plain=1#L134 -->

To release a new version of the CLI, follow the steps below:

> [!WARNING]
> You need to have write permission to this repo to release a new version

- [ ] Checkout `main` and pull the latest changes:

    ```sh
    git checkout main && git pull origin main
    ```

- [ ] In your local clone, create a new tag:

    ```sh
    git tag vX.Y.Z -m "Release vX.Y.Z"
    ```

- [ ] Push the tag to the repository:

    ```bash
    git push origin vX.Y.Z
    ```

    This will automatically start the [release workflow](./.github/workflows/release.yml) and create a draft release in the repo's [Releases](https://github.com/Layr-Labs/avs-devnet/releases) section with all the required binaries and assets.

- [ ] Check the release notes, add any notable changes, and publish the release.
