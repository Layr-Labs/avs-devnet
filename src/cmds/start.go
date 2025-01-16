package cmds

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
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

// Starts the devnet with the given context.
func StartCmd(ctx *cli.Context) error {
	pkgName := flags.KurtosisPackageFlag.Get(ctx)
	devnetName, configPath, err := parseArgs(ctx)
	if err != nil {
		return cli.Exit(err, 1)
	}
	configPath, err = filepath.Abs(configPath)
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
	workingDir := filepath.Dir(configPath)
	opts := StartOptions{
		KurtosisPackageUrl: pkgName,
		DevnetName:         devnetName,
		WorkingDir:         workingDir,
		DevnetConfig:       devnetConfig,
	}
	err = Start(ctx.Context, opts)
	if err != nil {
		return cli.Exit(err, 4)
	}
	return nil
}

// Options accepted by Start.
type StartOptions struct {
	// URL of the kurtosis package to run
	KurtosisPackageUrl string
	// Name of the devnet
	DevnetName string
	// Path to the working directory for the devnet.
	// Used when resolving relative paths.
	WorkingDir string
	// Devnet configuration
	DevnetConfig config.DevnetConfig
}

// Starts the devnet with the given context.
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

	err = buildDockerImages(opts.WorkingDir, opts.DevnetConfig)
	if err != nil {
		return fmt.Errorf("failed when building images: %w", err)
	}

	err = uploadLocalRepos(opts.WorkingDir, opts.DevnetConfig, enclaveCtx)
	if err != nil {
		return fmt.Errorf("failed when uploading local repos: %w", err)
	}

	err = uploadStaticFiles(opts.WorkingDir, opts.DevnetConfig, enclaveCtx)
	if err != nil {
		return fmt.Errorf("failed when uploading static files: %w", err)
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

// Uploads the local repositories to the enclave.
func uploadLocalRepos(dirContext string, config config.DevnetConfig, enclaveCtx *enclaves.EnclaveContext) error {
	for _, deployment := range config.Deployments {
		if deployment.Repo == "" {
			continue
		}
		repoUrl, err := url.Parse(deployment.Repo)
		if err != nil {
			return fmt.Errorf("repo '%s' is invalid: %w", deployment.Repo, err)
		}
		if !isLocalUrl(repoUrl.Scheme) {
			continue
		}
		absPath := ensureAbs(dirContext, repoUrl.Path)
		err = uploadLocalRepo(deployment, absPath, enclaveCtx)
		if err != nil {
			return fmt.Errorf("local repo '%s' uploading failed: %w", absPath, err)
		}
	}
	return nil
}

// TODO: to avoid having foundry as a dependency, we should use it via docker.
func uploadLocalRepo(deployment config.Deployment, repoPath string, enclaveCtx *enclaves.EnclaveContext) error {
	scriptPath := deployment.GetScriptPath()
	scriptOrigin := filepath.Join(repoPath, deployment.ContractsPath, scriptPath)

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

	originContractsDir := filepath.Join(repoPath, deployment.ContractsPath)
	// Install deps
	output, err := executeCmdInsideDir(originContractsDir, "forge install").CombinedOutput()
	if err != nil {
		return fmt.Errorf("forge install failed: %w, with output: %s", err, string(output))
	}
	// Flatten the script into a single file before upload
	flattenCmd := fmt.Sprintf("forge flatten -o %s %s", scriptDestination, scriptOrigin)
	output, err = executeCmdInsideDir(originContractsDir, flattenCmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("script flattening failed: %w, with output: %s", err, string(output))
	}

	// Copy the foundry config inside the contracts dir
	foundryConfigRelPath := filepath.Join(deployment.ContractsPath, "foundry.toml")
	src := filepath.Join(repoPath, foundryConfigRelPath)
	dst := filepath.Join(outputDir, foundryConfigRelPath)
	err = fileCopy(src, dst)
	if err != nil {
		return fmt.Errorf("failed when copying foundry.toml: %w", err)
	}

	// Upload the file to the enclave
	artifactName := deployment.Name + "-script"
	_, _, err = enclaveCtx.UploadFiles(outputDir, artifactName)
	if err != nil {
		return fmt.Errorf("file uploading failed: %w", err)
	}
	return nil
}

func uploadStaticFiles(dirContext string, config config.DevnetConfig, enclaveCtx *enclaves.EnclaveContext) error {
	for artifactName, artifactDetails := range config.Artifacts {
		numStaticFiles := 0
		numTemplates := 0
		for _, fileAttrs := range artifactDetails.Files {
			if fileAttrs.StaticFile != nil {
				numStaticFiles += 1
			} else if fileAttrs.Template != nil {
				numTemplates += 1
			} else {
				return errors.New("artifact must have either a static file or a template")
			}
		}
		if numStaticFiles > 0 && numTemplates > 0 {
			return errors.New("artifacts with both static files and templates are not yet supported")
		}
		// Skip artifacts with templates only
		if numStaticFiles == 0 {
			continue
		}

		// Output files in a temp dir
		outputDir, err := os.MkdirTemp(os.TempDir(), "avs-devnet-")
		if err != nil {
			return fmt.Errorf("tempdir creation failed: %w", err)
		}
		defer os.RemoveAll(outputDir)
		for outFileName, fileAttrs := range artifactDetails.Files {
			rawUrl := *fileAttrs.StaticFile
			destinationFilePath := filepath.Join(outputDir, outFileName)
			srcUrl, err := url.Parse(*fileAttrs.StaticFile)
			if err != nil {
				return fmt.Errorf("url '%s' is invalid: %w", rawUrl, err)
			}
			if isLocalUrl(srcUrl.Scheme) {
				// Copy the file to the temp dir
				originFilePath := ensureAbs(dirContext, rawUrl)
				err = fileCopy(originFilePath, destinationFilePath)
				if err != nil {
					return fmt.Errorf("failed when copying file: %w", err)
				}
			} else {
				// The file is remote. Download the file
				// 1. do GET request
				resp, err := http.Get(rawUrl)
				if err != nil {
					return fmt.Errorf("failed HTTP GET request: %w", err)
				}
				defer resp.Body.Close()
				// 2. check status code
				if resp.StatusCode < 200 && resp.StatusCode >= 300 {
					return fmt.Errorf("GET request failed with status code: %d", resp.StatusCode)
				}

				// 3. dump response to file
				dstFile, err := os.Create(destinationFilePath)
				if err != nil {
					return fmt.Errorf("failed to create file: %w", err)
				}
				defer dstFile.Close()

				_, err = io.Copy(dstFile, resp.Body)
				if err != nil {
					return fmt.Errorf("failed when downloading file: %w", err)
				}
			}
		}
		// Upload temp dir to enclave
		_, _, err = enclaveCtx.UploadFiles(outputDir, artifactName)
		if err != nil {
			return fmt.Errorf("file uploading failed: %w", err)
		}
	}
	return nil
}

// Builds the local docker images for the services in the configuration.
// Starts multiple builds in parallel.
func buildDockerImages(baseDir string, config config.DevnetConfig) error {
	errChan := make(chan error)
	numBuilds := 0
	for _, service := range config.Services {
		if service.BuildContext != nil {
			numBuilds += 1
			buildContext := ensureAbs(baseDir, *service.BuildContext)
			go func() {
				errChan <- buildWithDocker(service.Image, buildContext, service.BuildFile)
			}()
		} else if service.BuildCmd != nil {
			numBuilds += 1
			go func() {
				errChan <- buildWithCustomCmd(service.Image, baseDir, *service.BuildCmd)
			}()
		}
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
func buildWithCustomCmd(imageName, baseDir, buildCmd string) error {
	cmd := executeCmdInsideDir(baseDir, buildCmd)
	fmt.Println("Building image", imageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("building image '%s' failed: %w\n%s", imageName, err, output)
	}
	return nil
}

func ensureAbs(baseDir string, path string) string {
	absPath := path
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(baseDir, absPath)
	}
	return absPath
}

func executeCmdInsideDir(dir, cmd string) *exec.Cmd {
	fullCmd := fmt.Sprintf("cd %s && %s", dir, cmd)
	return exec.Command("sh", "-c", fullCmd)
}

func fileCopy(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer srcFile.Close()
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}
	return nil
}

func isLocalUrl(scheme string) bool {
	// If 'repo' starts with file:// or is without a scheme, it's a local repo
	return scheme == "file" || scheme == ""
}
