package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/taichi/javcli/pkg/javdb"
)

var detailCmd = &cobra.Command{
	Use:   "detail [code]",
	Short: "Get movie detail",
	Long:  `Get detail for the given code (including magnets). Output is JSON.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withClient(cmd, func(ctx context.Context, c *javdb.Client) (any, error) {
			return c.GetDetail(ctx, args[0])
		})
	},
}

func init() {
	rootCmd.AddCommand(detailCmd)
}
