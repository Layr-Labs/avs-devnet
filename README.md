# EigenLayer AVS Devnet

> [!WARNING]
> This repo is a Work-In-Progress. Use with caution!

*AvsDevnet* is a library and CLI tool to start local devnets with specific operator states.
We expect the library to be commonly used in place of mocks for automated testing of specific situations.
The CLI tool, on the other hand, should be used in place of anvil-like solutions for end-to-end testing.

## Features

### One line devnet setup

Currently, to have a local devnet with EigenLayer contracts deployed, we need to deploy them manually or build our own scripts.
This also includes deploying all of our AVS contracts.
With AvsDevnet we could make this as simple as a one line command.

### Extensively configurable

By having lots of tuning parameters for operators we can simulate complex situations.
We’re going to start operator registration and stakes setup only, but a lot of this could be extended in the future.

### Usable as a testing library

Being able to use it on unit tests will make automated testing easier.
With this, users won’t need to run complex setups before their tests.
They can just use the library and set the initial required state.
