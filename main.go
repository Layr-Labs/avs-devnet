package main

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"os/exec"
	"time"

	"github.com/Layr-Labs/eigensdk-go/chainio/clients/elcontracts"
	"github.com/Layr-Labs/eigensdk-go/chainio/clients/wallet"
	"github.com/Layr-Labs/eigensdk-go/chainio/txmgr"
	sdklogging "github.com/Layr-Labs/eigensdk-go/logging"
	"github.com/Layr-Labs/eigensdk-go/signerv2"
	"github.com/containerd/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func main() {
	// err := os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	// if err != nil {
	// 	log.L.Fatalf("err %v", err)
	// }
	anvilC, err := StartAnvilContainer()
	if err != nil {
		log.L.Error("erro = ", err)
	}
	endpoint, err := anvilC.Endpoint(context.Background(), "http")
	if err != nil {
		log.L.Error("error = ", err)
	}
	log.L.Info("anvil endpoint = ", endpoint)
	// endpoint := "http://localhost:8545"
	// fmt.Println("sleeping...")
	time.Sleep(5 * time.Second)

	DeployEigenLayerContracts(endpoint)

	time.Sleep(5 * time.Second)
	deploymentData := "eigenlayer-contracts/script/output/devnet/local_from_scratch_deployment_data.json"
	delegationManagerAddr, stderr, err := Shellout("jq -r .addresses.delegationManager " + deploymentData)
	if err != nil {
		log.L.Error("Failed to get delegation manager address, error = ", stderr)
		return
	}

	avsDirectoryAddr, stderr, err := Shellout("jq -r .addresses.avsDirectory " + deploymentData)
	if err != nil {
		log.L.Error("Failed to get avs directory address, error = ", stderr)
	}
	fmt.Println("delegationManagerAddr = ", delegationManagerAddr)
	fmt.Println("avsDirectoryAddr = ", avsDirectoryAddr)
	RegisterOperatorWithEigenLayer(endpoint, common.HexToAddress(delegationManagerAddr), common.HexToAddress(avsDirectoryAddr))
	time.Sleep(99999999 * time.Second)
}

func RegisterOperatorWithEigenLayer(anvilEndpoint string, delegationManagerAddr, avsDirectoryAddr common.Address) {
	client, err := ethclient.Dial(anvilEndpoint)
	fmt.Println("client = ", client)
	ecdsaPrivateKeyString := "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	ecdsaPrivateKey, err := crypto.HexToECDSA(ecdsaPrivateKeyString)

	chainId := big.NewInt(int64(31337))
	signerV2, _, err := signerv2.SignerFromConfig(signerv2.Config{PrivateKey: ecdsaPrivateKey}, chainId)
	if err != nil {
		panic(err)
	}
	operatorAddress := common.HexToAddress("f39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	logger, err := sdklogging.NewZapLogger("development")
	if err != nil {
		// return nil, err
		log.L.Error("failed to create logger, error = ", err)
		return
	}
	wallet, err := wallet.NewPrivateKeyWallet(client, signerV2, operatorAddress, logger)
	if err != nil {
		panic(err)
	}

	txMgr := txmgr.NewSimpleTxManager(
		wallet,
		client,
		logger,
		operatorAddress,
	)
	elWriter, err := elcontracts.BuildELChainWriter(
		delegationManagerAddr,
		avsDirectoryAddr,
		client,
		logger,
		nil, // eigenMetrics metrics.Metrics,
		txMgr,
	)
	if err != nil {
		log.L.Error("failed to create writer, error = ", err)
	}
	fmt.Println("writer = ", elWriter)
	// elWriter.RegisterAsOperator()
}

func DeployEigenLayerContracts(endpoint string) {
	log.L.Info("Building the contracts...")
	stdout, stderr, err := Shellout("cd eigenlayer-contracts && forge install")
	if err != nil {
		log.L.Info("failed to run forge install, error = ", stderr)
		return
	}
	log.L.Info(stdout)
	log.L.Info("Deploying the contracts...")
	scriptCommand := "cd eigenlayer-contracts && forge script script/deploy/local/Deploy_From_Scratch.s.sol --rpc-url " + endpoint + " --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 --broadcast --sig \"run(string memory configFile)\" -- local/deploy_from_scratch.anvil.config.json"
	stdout, stderr, err = Shellout(scriptCommand)
	if err != nil {
		log.L.Info("failed to deploy eigenlayer contracts, error = ", stderr)
		return
	}
	log.L.Info("stdout = ", stdout)
	deploymentData := "eigenlayer-contracts/script/output/devnet/local_from_scratch_deployment_data.json"
	log.L.Info("Deployed EigenLayer contracts successfully")
	stdout, stderr, err = Shellout("cat " + deploymentData)
	if err != nil {
		log.L.Error("error = ", err)
	} else {
		log.L.Info("Contracts were deployed to:\n", stdout)
	}
}

const ShellToUse = "bash"

func Shellout(command string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(ShellToUse, "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func StartAnvilContainer() (testcontainers.Container, error) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "ghcr.io/foundry-rs/foundry:nightly-3abac322efdb69e27b6fe8748b72754ae878f64d@sha256:871b66957335636a02c6c324c969db9adb1d6d64f148753c4a986cf32a40dc3c",
		Entrypoint:   []string{"anvil"},
		Cmd:          []string{"--host", "0.0.0.0", "--base-fee", "0", "--gas-price", "0"},
		ExposedPorts: []string{"8545/tcp"},
		WaitingFor:   wait.ForLog("Listening on"),
		Name:         "anvil",
	}

	anvilC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
		Reuse:            true,
	})
	if err != nil {
		return nil, err
	}

	anvilC.Start(ctx)

	return anvilC, nil
}
