package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/taichi/javcli/pkg/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Manage JavCLI configuration (cookies, proxy, locale).`,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long:  `Set a configuration value. Supported keys: cookies, proxy, locale.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if err := cfg.Set(key, value); err != nil {
			return err
		}
		if err := cfg.Save(); err != nil {
			return err
		}
		return writeJSON(cmd.OutOrStdout(), map[string]string{
			"message": fmt.Sprintf("Successfully set %s", key),
			"key":     key,
			"value":   value,
		})
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Long:  `Get a configuration value. Supported keys: cookies, proxy, locale.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		value, err := cfg.Get(key)
		if err != nil {
			return err
		}
		return writeJSON(cmd.OutOrStdout(), map[string]string{
			"key":   key,
			"value": value,
		})
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration",
	Long:  `List all configuration values.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		return writeJSON(cmd.OutOrStdout(), cfg.ToMap())
	},
}

var configUnsetCmd = &cobra.Command{
	Use:   "unset <key>",
	Short: "Unset a configuration value",
	Long:  `Unset (remove) a configuration value. Supported keys: cookies, proxy, locale.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if err := cfg.Set(key, ""); err != nil {
			return err
		}
		if err := cfg.Save(); err != nil {
			return err
		}
		return writeJSON(cmd.OutOrStdout(), map[string]string{
			"message": fmt.Sprintf("Successfully unset %s", key),
			"key":     key,
		})
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show the config file path",
	Long:  `Show the path to the configuration file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := config.GetConfigPath()
		if err != nil {
			return err
		}
		return writeJSON(cmd.OutOrStdout(), map[string]string{"path": path})
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configUnsetCmd)
	configCmd.AddCommand(configPathCmd)
}
