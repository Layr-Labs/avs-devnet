package cmds

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Layr-Labs/avs-devnet/src/config"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli/v2"
)

func printAddresses(args []string, devnetName string) error {
	failed := false
	cacheDir, err := os.MkdirTemp(os.TempDir(), ".devnet_cache")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(cacheDir) }()
	cached := make(map[string]string)

	for _, arg := range args {
		// contract name is like "artifact-name:contract-name"
		path := strings.Split(arg, ":")
		// TODO: assume a length of 1 means it's just the contract name
		if len(path) > 2 || len(path) == 1 {
			fmt.Println("Invalid contract name: " + arg)
			failed = true
			continue
		}
		artifactName := path[0]
		contractName := path[1]
		file, ok := cached[artifactName]
		if !ok {
			readFile, err := readJsonArtifact(cacheDir, devnetName, artifactName)
			if err != nil {
				fmt.Println("Error reading artifact", artifactName+":", err)
				failed = true
				readFile = ""
			}
			file = string(readFile)
			cached[artifactName] = file
		}
		output, ok := readArtifact(file, contractName)
		if !ok {
			fmt.Println("Error getting", arg)
			failed = true
			continue
		}
		fmt.Println(strings.TrimSpace(output))
	}
	if failed {
		return errors.New("failed to get some addresses")
	}
	return nil
}

func readArtifact(file string, contractName string) (string, bool) {
	var jsonPath string
	if strings.HasPrefix(contractName, ".") {
		// This uses the absolute path
		jsonPath = "addresses" + contractName + "|@pretty"
	} else if contractName != "" {
		// This searches for `contractName` inside the json
		// Since there are multiple results, `|0` is used to get the first one
		jsonPath = "@dig:" + contractName + "|0|@pretty"
	} else {
		// This just prints the whole json
		jsonPath = "@pretty"
	}
	res := gjson.Get(string(file), jsonPath)
	if !res.Exists() {
		return "", false
	}
	return res.String(), true
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

func readJsonArtifact(cacheDir string, devnetName string, artifactName string) (string, error) {
	err := exec.Command("sh", "-c", "cd "+cacheDir+" && kurtosis files download "+devnetName+" "+artifactName).Run()
	if err != nil {
		return "", err
	}
	artifactPath := filepath.Join(cacheDir, artifactName)
	dirEntry, err := os.ReadDir(artifactPath)
	if err != nil {
		return "", err
	}
	var readFile []byte
	for _, entry := range dirEntry {
		if entry.IsDir() {
			continue
		}
		fileName := entry.Name()
		if filepath.Ext(fileName) != ".json" {
			continue
		}
		readFile, err = os.ReadFile(filepath.Join(artifactPath, fileName))
		if err != nil {
			return "", err
		}
		break
	}
	return string(readFile), nil
}

func parseArgs(ctx *cli.Context) (string, string, error) {
	args := ctx.Args()
	if args.Len() > 1 {
		return "", "", errors.New("expected exactly 1 argument: <config-file>")
	}
	configFileName := args.First()
	var devnetName string
	if configFileName == "" {
		configFileName = "devnet.yaml"
		devnetName = "devnet"
	} else {
		name, err := nameFromConfigFile(configFileName)
		if err != nil {
			return "", "", err
		}
		devnetName = name
	}
	return configFileName, devnetName, nil
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !errors.Is(err, os.ErrNotExist)
}

func nameFromConfigFile(fileName string) (string, error) {
	name := filepath.Base(fileName)
	name = strings.Split(name, ".")[0]
	name = strings.ReplaceAll(name, "_", "-")
	matches, err := regexp.MatchString("^[-A-Za-z0-9]{1,60}$", name)
	if err != nil {
		// Error in regex pattern
		panic(err)
	}
	if !matches {
		return "", errors.New("Invalid devnet name: " + name)
	}
	return name, nil
}

func kurtosisRun(args ...string) error {
	cmd := exec.Command("kurtosis", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}
