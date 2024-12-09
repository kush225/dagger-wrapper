package main

import (
	"context"
	"fmt"
	"os"

	"dagger.io/dagger"
	"gopkg.in/yaml.v3"

	"github.com/kush225/dagger-wrapper/secrets"
)

type Config struct {
	SetupCommands [][]string `yaml:"setup_commands"`
	Steps []struct {
		Name            string            `yaml:"name"`
		BaseImage       string            `yaml:"base_image"`
		Workdir         string            `yaml:"workdir"`
		SourceDirectory string            `yaml:"source_directory"`
		Secrets         map[string]string `yaml:"secrets"`
		Commands        [][]string        `yaml:"commands"`
		EnvVariable		map[string]string `yaml:"variables"`
	} `yaml:"steps"`
	EnvVariable		map[string]string `yaml:"variables"`
}

func main() {
	ctx := context.Background()

	configPath := "dagger-config.yaml"
	config, err := loadConfig(configPath)
	if err != nil {
		fmt.Println("Error loading config:", err)
		os.Exit(1)
	}

	err = runSteps(ctx, config)
	if err != nil {
		fmt.Println("Error during steps execution:", err)
		os.Exit(1)
	}
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func runSteps(ctx context.Context, config *Config) error {
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
	if err != nil {
		return err
	}
	defer client.Close()

	for _, step := range config.Steps {
		fmt.Printf("Running step: %s\n", step.Name)
		secretsMap, err := secrets.FetchSecrets(ctx, step.Secrets)
		if err != nil {
			return fmt.Errorf("failed to fetch secrets for step %s: %v", step.Name, err)
		}

		container := client.Container().
			From(step.BaseImage).
			WithWorkdir(step.Workdir)

		for name, value := range config.EnvVariable{
			container = container.WithEnvVariable(name, value)
		}
	
		if step.SourceDirectory != "" {
			sourceDir := client.Host().Directory(step.SourceDirectory)
			container = container.WithDirectory(step.Workdir, sourceDir)
		}

		for name, value := range secretsMap {
			container = container.WithEnvVariable(name, value)
		}

		for name, value := range step.EnvVariable{
			container = container.WithEnvVariable(name, value)
		}

		for _, cmd := range config.SetupCommands {
			container = container.WithExec(cmd)
		}

		for _, cmd := range step.Commands {
			container = container.WithExec(cmd)
		}

		if _, err := container.ExitCode(ctx); err != nil {
			return fmt.Errorf("step %s failed: %v", step.Name, err)
		}

		fmt.Printf("Step %s completed successfully.\n", step.Name)
	}
	return nil
}
