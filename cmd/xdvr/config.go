package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newConfigCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configure your DSS instance",
	}

	cmd.AddCommand(newConfigGetCMD())
	cmd.AddCommand(newConfigSetCMD())
	cmd.AddCommand(newConfigShowCMD())

	return cmd
}

func newConfigGetCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get",
		Short:   "Get a config value",
		Example: "  xdvr config get apiKey",
		Args:    cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			switch args[0] {
			case "apiKey", "apikey":
				fmt.Printf("apiKey: %v\n", cfg.APIKey)
			case "autoDelete", "autodelete":
				fmt.Printf("autoDelete: %v\n", cfg.AutoDelete)
			case "savePath", "savepath":
				fmt.Printf("savePath: %v\n", cfg.SavePath)
			default:
				log.Fatal().Msg("unknown config key")
			}
		},
	}

	return cmd
}

func newConfigSetCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "set",
		Short:   "Set a config value",
		Example: "  xdvr config set apiKey your-api-key",
		Args:    cobra.ExactArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			switch args[0] {
			case "apiKey", "apikey":
				cfg.APIKey = args[1]
			case "autoDelete", "autodelete":
				cfg.AutoDelete = cast.ToBool(args[1])
			case "savePath", "savepath":
				// Check that the path exists.
				absolutePath, err := filepath.Abs(args[1])
				if err != nil {
					log.Fatal().Err(err).Msg("Failed to find absolute path")
				}

				absolutePath = filepath.Clean(absolutePath)

				info, err := os.Stat(absolutePath)
				if err != nil {
					if os.IsNotExist(err) {
						log.Fatal().Err(err).Msg("Path does not exist")
					}
					return
				}

				if !info.IsDir() {
					log.Fatal().Msg("Path is not a directory")
				}

				cfg.SavePath = absolutePath
			default:
				log.Fatal().Msg("Unknown config key")
			}

			if err := cfg.Save(); err != nil {
				log.Fatal().Err(err).Msg("Unable to save config")
			}

			log.Info().Msg("Config updated")
		},
	}

	return cmd
}

func newConfigShowCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show",
		Short:   "Print full config",
		Example: "  xdvr config show",
		Args:    cobra.ExactArgs(0),
		Run: func(_ *cobra.Command, _ []string) {
			output, err := yaml.Marshal(cfg)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to marshal YAML config")
			}

			fmt.Print(string(output))
		},
	}

	return cmd
}

type Config struct {
	dir        string
	path       string
	APIKey     string `json:"apiKey" yaml:"apiKey"`
	AutoDelete bool   `json:"autoDelete" yaml:"autoDelete"`
	SavePath   string `json:"savePath" yaml:"savePath"`
}

func newConfig(directory string) (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("find home directory: %w", err)
	}

	config := Config{
		dir:        directory,
		path:       filepath.Join(directory, "config.yml"),
		APIKey:     "",
		AutoDelete: false,
		SavePath:   filepath.Join(homeDir, "xbox-dvr"),
	}

	if err = config.Load(); err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *Config) Load() error {
	if _, err := os.Stat(c.path); os.IsNotExist(err) {
		if err = os.MkdirAll(c.dir, os.ModePerm); err != nil {
			log.Fatal().Err(err).Msg("Failed to create config directory")
		}

		if err = c.Save(); err != nil {
			return err
		}
	}

	configData, err := os.ReadFile(c.path)
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to read config file")
	}

	if err = yaml.Unmarshal(configData, &c); err != nil {
		log.Fatal().Err(err).Msg("Unable to unmarshal config values")
	}

	if val, ok := os.LookupEnv("apiKey"); ok {
		c.APIKey = val
	}

	if val, ok := os.LookupEnv("autoDelete"); ok {
		c.AutoDelete = cast.ToBool(val)
	}

	if val, ok := os.LookupEnv("savePath"); ok {
		c.SavePath = val
	}

	return nil
}

func (c *Config) Save() error {
	configData, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err = os.WriteFile(c.path, configData, os.ModePerm); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}
