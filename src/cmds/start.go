package cmds

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
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

	if err := buildDockerImages(config); err != nil {
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
	alreadyUploaded := make(map[string]bool)

	for _, deployment := range config.Deployments {
		if deployment.Repo == "" {
			continue
		}
		repoUrl, err := url.Parse(deployment.Repo)
		if err != nil {
			return err
		}
		if repoUrl.Scheme != "file" && repoUrl.Scheme != "" {
			continue
		}
		if alreadyUploaded[repoUrl.Path] {
			continue
		}
		path := repoUrl.Path
		err = uploadLocalRepo(deployment, path, enclaveCtx)
		if err != nil {
			return err
		}
		alreadyUploaded[repoUrl.Path] = true
	}
	return nil
}

func uploadLocalRepo(deployment config.Deployment, path string, enclaveCtx *enclaves.EnclaveContext) error {
	// Upload the file with the path as the name
	_, _, err := enclaveCtx.UploadFiles(path, path)
	if err != nil {
		return err
	}
	return nil
}

// Builds the local docker images for the services in the configuration
func buildDockerImages(config config.DevnetConfig) error {
	errChan := make(chan error)
	numBuilds := 0
	for _, service := range config.Services {
		if service.BuildContext == nil {
			continue
		}
		image := service.Image
		cmdArgs := []string{"docker", "build", *service.BuildContext, "-t", image}
		if service.BuildFile != nil {
			cmdArgs = append(cmdArgs, "-f", *service.BuildFile)
		}
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		fmt.Println("Building image", image)
		go func() {
			output, err := cmd.CombinedOutput()
			if err != nil {
				err = fmt.Errorf("building image '%s' failed: %w\n%s", image, err, output)
			}
			errChan <- err
		}()
		numBuilds += 1
	}
	errs := make([]error, numBuilds)
	for range numBuilds {
		errs = append(errs, <-errChan)
	}
	return errors.Join(errs...)
}
