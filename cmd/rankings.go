package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/taichi/javcli/pkg/javdb"
)

var (
	rankingPeriod string
	rankingType   string
)

var rankingsCmd = &cobra.Command{
	Use:   "rankings",
	Short: "Show rankings",
	Long:  `Show daily/weekly/monthly rankings. Output is JSON.`,
	Run: func(cmd *cobra.Command, args []string) {
		client := javdb.NewClient()
		results, err := client.GetRankings(rankingPeriod, rankingType)
		if err != nil {
			outputErrorJSON(cmd, err)
			return
		}

		jsonData, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			outputErrorJSON(cmd, err)
			return
		}

		fmt.Println(string(jsonData))
	},
}

func init() {
	rootCmd.AddCommand(rankingsCmd)

	rankingsCmd.Flags().StringVarP(&rankingPeriod, "period", "p", "daily", "Period: daily, weekly, monthly")
	rankingsCmd.Flags().StringVarP(&rankingType, "type", "t", "censored", "Type: censored, uncensored, western, fc2")
}
