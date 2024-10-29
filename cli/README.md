# AvsDevnet CLI

A wrapper over the [Kurtosis CLI](https://docs.kurtosis.com/cli).

## Building the program

```sh
go build -o avs-devnet cmd/main.go
```

## Starting the devnet

```sh
./avs-devnet run -c <config-path>
```

## Stopping the devnet

```sh
./avs-devnet clean -c <config-path>
```
