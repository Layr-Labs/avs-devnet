package cmds

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Layr-Labs/avs-devnet/src/cmds/flags"
	"github.com/Layr-Labs/avs-devnet/src/config"
	"github.com/Layr-Labs/avs-devnet/src/kurtosis"
	"github.com/Layr-Labs/avs-devnet/src/kurtosis/progress_reporters"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/starlark_run_config"
	"github.com/urfave/cli/v2"
)

// Starts the devnet with the given context
func Start(ctx *cli.Context) error {
	fmt.Println("Starting devnet...")
	pkgName := ctx.String(flags.KurtosisPackageFlag.Name)
	configPath, devnetName, err := parseArgs(ctx)
	if err != nil {
		return cli.Exit(err, 1)
	}
	if !fileExists(configPath) {
		return cli.Exit("Config file doesn't exist: "+configPath, 2)
	}

	if err := startDevnet(ctx, pkgName, devnetName, configPath); err != nil {
		return cli.Exit(err, 3)
	}
	return nil
}

// Starts the devnet with the given configuration
func startDevnet(ctx *cli.Context, pkgName, devnetName string, configPath string) error {
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	config, err := config.Unmarshal(configBytes)
	if err != nil {
		return err
	}

	kurtosisCtx, err := kurtosis.InitKurtosisContext()
	if err != nil {
		return err
	}
	if kurtosisCtx.EnclaveExists(ctx.Context, devnetName) {
		return errors.New("devnet already running")
	}
	enclaveCtx, err := kurtosisCtx.CreateEnclave(ctx.Context, devnetName)
	if err != nil {
		return err
	}

	err = buildDockerImages(config)
	if err != nil {
		return err
	}

	err = uploadLocalRepos(config, enclaveCtx)
	if err != nil {
		return err
	}

	starlarkConfig := starlark_run_config.NewRunStarlarkConfig()
	starlarkConfig.SerializedParams = string(configBytes)

	var responseChan chan progress_reporters.KurtosisResponse
	// TODO: use cancel func if needed
	// var cancel context.CancelFunc
	if strings.HasPrefix(pkgName, "github.com/") {
		responseChan, _, err = enclaveCtx.RunStarlarkRemotePackage(ctx.Context, pkgName, starlarkConfig)
	} else {
		responseChan, _, err = enclaveCtx.RunStarlarkPackage(ctx.Context, pkgName, starlarkConfig)
	}
	if err != nil {
		return err
	}

	reporter := progress_reporters.NewProgressBarReporter()
	return progress_reporters.ReportProgress(reporter, responseChan)
}

// Uploads the local repositories to the enclave
func uploadLocalRepos(config config.DevnetConfig, enclaveCtx *enclaves.EnclaveContext) error {
	for _, deployment := range config.Deployments {
		if deployment.Repo == "" {
			continue
		}
		repoUrl, err := url.Parse(deployment.Repo)
		if err != nil {
			return fmt.Errorf("repo '%s' is invalid: %w", deployment.Repo, err)
		}
		if repoUrl.Scheme != "file" && repoUrl.Scheme != "" {
			continue
		}
		err = uploadLocalRepo(deployment, repoUrl.Path, enclaveCtx)
		if err != nil {
			return fmt.Errorf("local repo '%s' uploading failed: %w", repoUrl.Path, err)
		}
	}
	return nil
}

func uploadLocalRepo(deployment config.Deployment, repoPath string, enclaveCtx *enclaves.EnclaveContext) error {
	scriptPath := deployment.GetScriptPath()
	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return err
	}
	scriptOrigin := filepath.Join(absRepoPath, deployment.ContractsPath, scriptPath)

	outputDir, err := os.MkdirTemp(os.TempDir(), "avs-devnet-")
	if err != nil {
		return fmt.Errorf("tempdir creation failed: %w", err)
	}
	defer os.RemoveAll(outputDir)

	scriptDestination := filepath.Join(outputDir, deployment.ContractsPath, scriptPath)

	err = os.MkdirAll(filepath.Dir(scriptDestination), 0700)
	if err != nil {
		return fmt.Errorf("output dir creation failed: %w", err)
	}

	if !fileExists(scriptOrigin) {
		return fmt.Errorf("file '%s' doesn't exist", scriptOrigin)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	err = os.Chdir(filepath.Join(absRepoPath, deployment.ContractsPath))
	if err != nil {
		return fmt.Errorf("chdir to contracts dir failed: %w", err)
	}
	// Install deps
	output, err := exec.Command("forge", "install").CombinedOutput()
	if err != nil {
		return fmt.Errorf("forge install failed: %w, with output: %s", err, string(output))
	}
	// Flatten the script into a single file
	output, err = exec.Command("forge", "flatten", "-o", scriptDestination, scriptOrigin).CombinedOutput()
	if err != nil {
		return fmt.Errorf("script flattening failed: %w, with output: %s", err, string(output))
	}

	foundryConfigRelPath := filepath.Join(deployment.ContractsPath, "foundry.toml")
	foundryConfig, err := os.ReadFile(filepath.Join(absRepoPath, foundryConfigRelPath))
	if err != nil {
		return fmt.Errorf("failed to read foundry.toml: %w", err)
	}
	file, err := os.Create(filepath.Join(outputDir, foundryConfigRelPath))
	if err != nil {
		return err
	}
	_, err = file.Write(foundryConfig)
	if err != nil {
		return err
	}
	file.Close()

	err = os.Chdir(cwd)
	if err != nil {
		return err
	}
	// Upload the file with the script path as the name
	artifactName := deployment.Name + "-script"
	_, _, err = enclaveCtx.UploadFiles(outputDir, artifactName)
	if err != nil {
		return fmt.Errorf("file uploading failed: %w", err)
	}
	return nil
}

// Builds the local docker images for the services in the configuration
func buildDockerImages(config config.DevnetConfig) error {
	errChan := make(chan error)
	numBuilds := 0
	for _, service := range config.Services {
		if service.BuildContext != nil {
			go func() {
				errChan <- buildWithDocker(service.Image, *service.BuildContext, service.BuildFile)
			}()
		} else if service.BuildCmd != nil {
			go func() {
				errChan <- buildWithCustomCmd(service.Image, *service.BuildCmd)
			}()
		}
		numBuilds += 1
	}
	errs := make([]error, numBuilds)
	for range numBuilds {
		errs = append(errs, <-errChan)
	}
	return errors.Join(errs...)
}

func buildWithDocker(imageName string, buildContext string, buildFile *string) error {
	cmdArgs := []string{"build", buildContext, "-t", imageName}
	if buildFile != nil {
		cmdArgs = append(cmdArgs, "-f", *buildFile)
	}
	cmd := exec.Command("docker", cmdArgs...)
	fmt.Println("Building image", imageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("building image '%s' failed: %w\n%s", imageName, err, output)
	}
	return nil
}

func buildWithCustomCmd(imageName string, buildCmd string) error {
	cmd := exec.Command("sh", "-c", buildCmd)
	fmt.Println("Building image", imageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("building image '%s' failed: %w\n%s", imageName, err, output)
	}
	return nil
}
