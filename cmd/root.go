package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/taichi/javcli/pkg/config"
)

var rootCmd = &cobra.Command{
	Use:   "javcli",
	Short: "JavDB CLI",
	Long:  `javcli is a CLI for searching and browsing the JavDB database. Output is always JSON.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"error": err.Error()})
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Load config and apply to environment variables
	// This allows config values to be used by all commands
	// Environment variables take precedence over config file
	cfg, err := config.Load()
	if err != nil {
		// Silently ignore config load errors during init
		// The config command will handle errors properly
		return
	}
	cfg.ApplyToEnv()
}
