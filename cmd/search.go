package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/taichi/javcli/pkg/javdb"
)

var searchCmd = &cobra.Command{
	Use:   "search [code or keyword]",
	Short: "Search by code or keyword",
	Long:  `Search by code or keyword and return JSON.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]

		client := javdb.NewClient()
		results, err := client.Search(query)
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
	rootCmd.AddCommand(searchCmd)
}
