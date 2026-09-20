package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/taichi/javcli/pkg/javdb"
)

var rankingsCmd = &cobra.Command{
	Use:   "rankings",
	Short: "Show rankings",
	Long:  `Show daily/weekly/monthly rankings. Output is JSON.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		period, err := cmd.Flags().GetString("period")
		if err != nil {
			return err
		}
		rankingType, err := cmd.Flags().GetString("type")
		if err != nil {
			return err
		}
		return withClient(cmd, func(ctx context.Context, c *javdb.Client) (any, error) {
			return c.GetRankings(ctx, period, rankingType)
		})
	},
}

func init() {
	rootCmd.AddCommand(rankingsCmd)
	rankingsCmd.Flags().StringP("period", "p", "daily", "Period: daily, weekly, monthly")
	rankingsCmd.Flags().StringP("type", "t", "censored", "Type: censored, uncensored, western, fc2")
}
