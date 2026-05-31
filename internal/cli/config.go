package cli

import (
	"fmt"
	"strconv"

	"github.com/Omotolani98/rekord/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newConfigCommand() *cobra.Command {
	var cfgPath string

	cmd := &cobra.Command{
		Use:   "config",
		Short: "View and edit rekord configuration",
	}
	cmd.PersistentFlags().StringVar(&cfgPath, "config", "rekord.yaml", "config file path")

	cmd.AddCommand(
		newConfigPathCommand(&cfgPath),
		newConfigViewCommand(&cfgPath),
		newConfigGetCommand(&cfgPath),
		newConfigSetCommand(&cfgPath),
	)
	return cmd
}

func newConfigPathCommand(cfgPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the config file path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), *cfgPath)
			return err
		},
	}
}

func newConfigViewCommand(cfgPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "view",
		Short: "Print the current configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}
			data, err := yaml.Marshal(cfg)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(data)
			return err
		},
	}
}

func newConfigGetCommand(cfgPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Print a config value (recording.stopKey, privacy.redact)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}
			val, err := configGet(cfg, args[0])
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), val)
			return err
		},
	}
}

func newConfigSetCommand(cfgPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value (recording.stopKey, privacy.redact)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}
			if err := configSet(&cfg, args[0], args[1]); err != nil {
				return err
			}
			if err := config.Save(*cfgPath, cfg); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s = %s\n", args[0], args[1])
			return err
		},
	}
}

func configGet(cfg config.Config, key string) (string, error) {
	switch key {
	case "recording.stopKey":
		if cfg.Recording.StopKey == "" {
			return defaultStopKey, nil
		}
		return cfg.Recording.StopKey, nil
	case "privacy.redact":
		return strconv.FormatBool(cfg.Privacy.Redact), nil
	default:
		return "", fmt.Errorf("unknown config key %q", key)
	}
}

func configSet(cfg *config.Config, key, value string) error {
	switch key {
	case "recording.stopKey":
		if _, _, err := parseStopKey(value); err != nil {
			return err
		}
		cfg.Recording.StopKey = value
	case "privacy.redact":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("privacy.redact must be true or false: %w", err)
		}
		cfg.Privacy.Redact = b
	default:
		return fmt.Errorf("unknown config key %q", key)
	}
	return nil
}
