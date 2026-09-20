package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/taichi/javcli/pkg/javdb"
)

var searchCmd = &cobra.Command{
	Use:   "search [code or keyword]",
	Short: "Search by code or keyword",
	Long:  `Search by code or keyword and return JSON.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withClient(cmd, func(ctx context.Context, c *javdb.Client) (any, error) {
			return c.Search(ctx, args[0])
		})
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
