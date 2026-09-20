package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/taichi/javcli/pkg/javdb"
)

var reviewsPage int

var reviewsCmd = &cobra.Command{
	Use:   "reviews [code]",
	Short: "List reviews for a code",
	Long:  `Get paginated review list for the given code. Output is JSON.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		code := args[0]

		client := javdb.NewClient()
		result, err := client.GetReviews(code, reviewsPage)
		if err != nil {
			outputErrorJSON(cmd, err)
			return
		}

		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			outputErrorJSON(cmd, err)
			return
		}

		fmt.Println(string(jsonData))
	},
}

func init() {
	rootCmd.AddCommand(reviewsCmd)
	reviewsCmd.Flags().IntVarP(&reviewsPage, "page", "p", 1, "Page number")
}
