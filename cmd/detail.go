package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/taichi/javcli/pkg/javdb"
)

var detailCmd = &cobra.Command{
	Use:   "detail [code]",
	Short: "Get movie detail",
	Long:  `Get detail for the given code (including magnets). Output is JSON.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		code := args[0]

		client := javdb.NewClient()
		detail, err := client.GetDetail(code)
		if err != nil {
			outputErrorJSON(cmd, err)
			return
		}

		jsonData, err := json.MarshalIndent(detail, "", "  ")
		if err != nil {
			outputErrorJSON(cmd, err)
			return
		}

		fmt.Println(string(jsonData))
	},
}

func init() {
	rootCmd.AddCommand(detailCmd)
}
