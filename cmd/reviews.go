package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/taichi/javcli/pkg/javdb"
)

var reviewsCmd = &cobra.Command{
	Use:   "reviews [code]",
	Short: "List reviews for a code",
	Long:  `Get paginated review list for the given code. Output is JSON.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		page, err := cmd.Flags().GetInt("page")
		if err != nil {
			return err
		}
		return withClient(cmd, func(ctx context.Context, c *javdb.Client) (any, error) {
			return c.GetReviews(ctx, args[0], page)
		})
	},
}

func init() {
	rootCmd.AddCommand(reviewsCmd)
	reviewsCmd.Flags().IntP("page", "p", 1, "Page number")
}
