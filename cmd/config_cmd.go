package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/sgaunet/retry/pkg/config"
	"github.com/spf13/cobra"
)

var (
	ErrConfigFileExists = errors.New("config file already exists (use --force to overwrite)")
	ErrNoConfigFile     = errors.New("no config file found")
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage retry configuration files",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a template retry.yaml in the current directory",
	RunE: func(_ *cobra.Command, _ []string) error {
		target := "retry.yaml"
		force, _ := configCmd.Flags().GetBool("force")

		if !force {
			if _, err := os.Stat(target); err == nil {
				return ErrConfigFileExists
			}
		}

		const configFilePerms = 0o644
		//nolint:gosec // Config template is not sensitive, 0644 allows team sharing
		if err := os.WriteFile(target, []byte(config.GenerateTemplate()), configFilePerms); err != nil {
			return fmt.Errorf("failed to write %s: %w", target, err)
		}

		fmt.Fprintf(os.Stderr, "Created %s\n", target)
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display effective merged configuration",
	RunE: func(_ *cobra.Command, _ []string) error {
		path := config.FindConfigFile(configFile)
		if path == "" {
			return ErrNoConfigFile
		}

		cfg, err := config.Load(path)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Config file: %s\n\n", path)
		fmt.Print(config.FormatEffective(cfg, profileName))
		return nil
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a configuration file",
	RunE: func(_ *cobra.Command, _ []string) error {
		path := config.FindConfigFile(configFile)
		if path == "" {
			return ErrNoConfigFile
		}

		cfg, err := config.Load(path)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("validation error: %w", err)
		}

		fmt.Fprintf(os.Stderr, "%s: valid\n", path)
		return nil
	},
}

func setupConfigCommands() {
	configCmd.PersistentFlags().Bool("force", false, "overwrite existing file")

	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configValidateCmd)

	rootCmd.AddCommand(configCmd)
}
