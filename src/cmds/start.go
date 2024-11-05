package cmds

import (
	"errors"
	"fmt"
	"net/url"
	"os/exec"

	"github.com/Layr-Labs/avs-devnet/src/cmds/flags"
	"github.com/Layr-Labs/avs-devnet/src/config"
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

	config, err := config.LoadFromPath(configPath)
	if err != nil {
		return cli.Exit(err, 2)
	}

	err = buildDockerImages(config)
	if err != nil {
		return cli.Exit(err, 3)
	}

	if err = kurtosisRun("enclave", "add", "--name", devnetName); err != nil {
		return cli.Exit(err, 4)
	}

	err = uploadLocalRepos(config, devnetName)
	if err != nil {
		return cli.Exit(err, 5)
	}

	return kurtosisRun("run", pkgName, "--enclave", devnetName, "--args-file", configPath)
}

func uploadLocalRepos(config config.DevnetConfig, devnetName string) error {
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
		if err := kurtosisRun("files", "upload", "--name", path, devnetName, path); err != nil {
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
