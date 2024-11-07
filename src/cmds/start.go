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
	"github.com/kurtosis-tech/kurtosis/api/golang/core/kurtosis_core_rpc_api_bindings"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/starlark_run_config"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
)

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

	var responseChan chan *kurtosis_core_rpc_api_bindings.StarlarkRunResponseLine
	// TODO: use cancel func if needed
	// var cancel context.CancelFunc
	// TODO: stream result lines and give user progress updates
	if strings.HasPrefix(pkgName, "github.com/") {
		responseChan, _, err = enclaveCtx.RunStarlarkRemotePackage(ctx.Context, pkgName, starlarkConfig)
	} else {
		responseChan, _, err = enclaveCtx.RunStarlarkPackage(ctx.Context, pkgName, starlarkConfig)
	}
	if err != nil {
		return err
	}

	// state := "validation"
	pb := newValidationProgressBar()
	if err = pb.RenderBlank(); err != nil {
		return err
	}
	for line := range responseChan {
		if line.GetProgressInfo() != nil {
			// It's a progress info
			// fmt.Println(line.GetProgressInfo())
			progressInfo := line.GetProgressInfo()
			if len(progressInfo.CurrentStepInfo) == 0 {
				continue
			}
			if progressInfo.CurrentStepInfo[0] == "Starting validation" {
				continue
			}
			fmt.Fprintln(os.Stderr, "progress: ", progressInfo)

			pb.Describe(progressInfo.CurrentStepInfo[0])
			detail := ""
			if len(progressInfo.CurrentStepInfo) > 1 {
				detail = progressInfo.CurrentStepInfo[1]
			}
			if err = pb.AddDetail(detail); err != nil {
				return err
			}
			if progressInfo.TotalSteps != 0 {
				if progressInfo.CurrentStepNumber >= progressInfo.TotalSteps {
					pb.Close()
					pb = newExecutionProgressBar(-1)
					if err = pb.RenderBlank(); err != nil {
						return err
					}
				} else {
					pb.ChangeMax(int(progressInfo.TotalSteps))
					if err = pb.Set(int(progressInfo.CurrentStepNumber)); err != nil {
						return err
					}
				}
			}
		}
		if line.GetInstruction() != nil {
			// It's an instruction
			fmt.Fprintln(os.Stderr, line.GetInstruction().Description)
		}
		if line.GetInfo() != nil {
			// It's an info
			fmt.Fprintln(os.Stderr, "INFO:", line.GetInfo().InfoMessage)
		}
		if line.GetWarning() != nil {
			// It's a warning
			fmt.Fprintln(os.Stderr, line.GetWarning().WarningMessage)
		}
		if line.GetInstructionResult() != nil {
			// It's an instruction result
			fmt.Fprintln(os.Stderr, line.GetInstructionResult())
		}
		if line.GetError() != nil {
			// It's an error
			// pb.Exit()
			fmt.Fprintln(os.Stderr, line.GetError())
		}
		if line.GetRunFinishedEvent() != nil {
			// It's a run finished event
			// pb.Close()
			fmt.Fprintln(os.Stderr, line.GetRunFinishedEvent())
		}
	}

	fmt.Println("Devnet started!")
	return err
}

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

func newValidationProgressBar() *progressbar.ProgressBar {
	pb := progressbar.NewOptions(
		-1,
		progressbar.OptionSetMaxDetailRow(1),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetElapsedTime(true),
		progressbar.OptionSetPredictTime(false),
		// TODO: use Stderr for progress bar
		// progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSetWriter(os.Stdout),
		progressbar.OptionShowCount(),
		// progressbar.OptionOnCompletion(func() {
		// 	fmt.Fprint(os.Stderr, "\n")
		// }),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionSetWidth(20),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionEnableColorCodes(true),
	)
	return pb
}

func newExecutionProgressBar(steps int) *progressbar.ProgressBar {
	pb := progressbar.NewOptions(
		steps,
		progressbar.OptionSetMaxDetailRow(2),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetElapsedTime(true),
		progressbar.OptionSetPredictTime(false),
		// TODO: use Stderr for progress bar
		// progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSetWriter(os.Stdout),
		progressbar.OptionShowCount(),
		// progressbar.OptionOnCompletion(func() {
		// 	fmt.Fprint(os.Stderr, "\n")
		// }),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionSetWidth(50),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionEnableColorCodes(true),
	)
	return pb
}
