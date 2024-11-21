package cmds

import (
	"context"
	"errors"
	"fmt"
	"net/url"
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
func StartCmd(ctx *cli.Context) error {
	pkgName := ctx.String(flags.KurtosisPackageFlag.Name)
	devnetName, configPath, err := parseArgs(ctx)
	if err != nil {
		return cli.Exit(err, 1)
	}
	if !fileExists(configPath) {
		return cli.Exit("Config file doesn't exist: "+configPath, 2)
	}
	devnetConfig, err := config.LoadFromPath(configPath)
	if err != nil {
		return cli.Exit(err, 3)
	}
	opts := StartOptions{
		KurtosisPackageUrl: pkgName,
		DevnetName:         devnetName,
		DevnetConfig:       devnetConfig,
	}
	err = Start(ctx.Context, opts)
	if err != nil {
		return cli.Exit(err, 4)
	}
	return nil
}

// Options accepted by Start
type StartOptions struct {
	KurtosisPackageUrl string
	DevnetName         string
	DevnetConfig       config.DevnetConfig
}

// Starts the devnet with the given context
func Start(ctx context.Context, opts StartOptions) error {
	kurtosisCtx, err := kurtosis.InitKurtosisContext()
	if err != nil {
		return fmt.Errorf("failed to initialize kurtosis context: %w", err)
	}
	if kurtosisCtx.EnclaveExists(ctx, opts.DevnetName) {
		return errors.New("devnet already running")
	}
	enclaveCtx, err := kurtosisCtx.CreateEnclave(ctx, opts.DevnetName)
	if err != nil {
		return fmt.Errorf("failed to create enclave: %w", err)
	}

	err = buildDockerImages(opts.DevnetConfig)
	if err != nil {
		return fmt.Errorf("failed when building images: %w", err)
	}

	err = uploadLocalRepos(opts.DevnetConfig, enclaveCtx)
	if err != nil {
		return fmt.Errorf("failed when uploading local repos: %w", err)
	}

	starlarkConfig := starlark_run_config.NewRunStarlarkConfig()
	starlarkConfig.SerializedParams = string(opts.DevnetConfig.Marshal())

	kurtosisPkg := opts.KurtosisPackageUrl
	if kurtosisPkg == "" {
		kurtosisPkg = flags.DefaultKurtosisPackage
	}

	fmt.Println("Starting devnet...")

	var responseChan chan progress_reporters.KurtosisResponse
	// TODO: use cancel func if needed
	// var cancel context.CancelFunc
	if strings.HasPrefix(kurtosisPkg, "github.com/") {
		responseChan, _, err = enclaveCtx.RunStarlarkRemotePackage(ctx, kurtosisPkg, starlarkConfig)
	} else {
		responseChan, _, err = enclaveCtx.RunStarlarkPackage(ctx, kurtosisPkg, starlarkConfig)
	}
	if err != nil {
		return fmt.Errorf("failed when running kurtosis package: %w", err)
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
		// If 'repo' starts with file:// or is without a scheme, it's a local repo
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

// Uploads the script of a single deployment from the repo at the given path to an enclave.
// The deployment script is flattened and uploaded with the deployment name suffixed with '-script'.
// The resulting artifact's structure is similar to the repo's structure, but with only the script and foundry config.
// TODO: to avoid having foundry as a dependency, we should use it via docker
func uploadLocalRepo(deployment config.Deployment, repoPath string, enclaveCtx *enclaves.EnclaveContext) error {
	scriptPath := deployment.GetScriptPath()
	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return err
	}
	scriptOrigin := filepath.Join(absRepoPath, deployment.ContractsPath, scriptPath)

	// Output files in a temp dir
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

	// Verify the script exists
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
	// Flatten the script into a single file before upload
	output, err = exec.Command("forge", "flatten", "-o", scriptDestination, scriptOrigin).CombinedOutput()
	if err != nil {
		return fmt.Errorf("script flattening failed: %w, with output: %s", err, string(output))
	}

	// Copy the foundry config inside the contracts dir
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
	// Upload the file to the enclave
	artifactName := deployment.Name + "-script"
	_, _, err = enclaveCtx.UploadFiles(outputDir, artifactName)
	if err != nil {
		return fmt.Errorf("file uploading failed: %w", err)
	}
	return nil
}

// Builds the local docker images for the services in the configuration.
// Starts multiple builds in parallel.
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
	// Check that all builds were successful and fail if not
	errs := make([]error, numBuilds)
	for range numBuilds {
		errs = append(errs, <-errChan)
	}
	return errors.Join(errs...)
}

// Builds a docker image with the given name from the given build context and (optional) file.
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

// Builds a docker image with the given name with a custom command.
// The command is executed inside a shell.
func buildWithCustomCmd(imageName string, buildCmd string) error {
	cmd := exec.Command("sh", "-c", buildCmd)
	fmt.Println("Building image", imageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("building image '%s' failed: %w\n%s", imageName, err, output)
	}
	return nil
}
