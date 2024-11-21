package cmds

import (
	"context"
	"errors"
	"fmt"
	"net/url"
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
		ShowProgressBar:    true,
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
	ShowProgressBar    bool
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

	var reporter progress_reporters.Reporter
	if opts.ShowProgressBar {
		reporter = progress_reporters.NewProgressBarReporter()
	}
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
		// Upload the file with the path as the name
		if _, _, err := enclaveCtx.UploadFiles(path, path); err != nil {
			return err
		}
		alreadyUploaded[repoUrl.Path] = true
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
