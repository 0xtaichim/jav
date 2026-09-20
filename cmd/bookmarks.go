package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/taichi/javcli/pkg/javdb"
)

var (
	bookmarkType string
	bookmarkPage int
)

var (
	bookmarkAddRemoveType string
	bookmarkAddRating     int
	bookmarkAddContent    string
)

var bookmarksCmd = &cobra.Command{
	Use:   "bookmarks",
	Short: "List bookmarks",
	Long:  `List want_watch, watched, or actors. Requires JAVDB_COOKIES. Output is JSON.`,
	Run: func(cmd *cobra.Command, args []string) {
		client := javdb.NewClient()
		results, err := client.GetBookmarks(bookmarkType, bookmarkPage)
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

var bookmarksAddCmd = &cobra.Command{
	Use:   "add [code]",
	Short: "Add bookmark",
	Long:  `Add want_watch: bookmarks add -t want_watch <code>. Add watched: bookmarks add -t watched <code> [--rating 1-5] [--content "comment"]`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := javdb.NewClient()
		code := args[0]
		switch bookmarkAddRemoveType {
		case "want_watch":
			if err := client.AddWantWatch(code); err != nil {
				outputErrorJSON(cmd, err)
				return
			}
			_ = outputJSON(cmd.OutOrStdout(), map[string]interface{}{"ok": true, "type": "want_watch", "code": code})
		case "watched":
			if err := client.AddWatched(code, bookmarkAddRating, bookmarkAddContent); err != nil {
				outputErrorJSON(cmd, err)
				return
			}
			_ = outputJSON(cmd.OutOrStdout(), map[string]interface{}{"ok": true, "type": "watched", "code": code})
		default:
			outputErrorJSON(cmd, fmt.Errorf("add only supports -t want_watch or -t watched"))
		}
	},
}

var bookmarksRemoveCmd = &cobra.Command{
	Use:   "remove [code]",
	Short: "Remove bookmark",
	Long:  `Remove want_watch: bookmarks remove -t want_watch <code>. Remove watched: bookmarks remove -t watched <code>`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := javdb.NewClient()
		code := args[0]
		switch bookmarkAddRemoveType {
		case "want_watch":
			if err := client.RemoveWantWatch(code); err != nil {
				outputErrorJSON(cmd, err)
				return
			}
			_ = outputJSON(cmd.OutOrStdout(), map[string]interface{}{"ok": true, "type": "want_watch", "code": code})
		case "watched":
			if err := client.RemoveWatched(code); err != nil {
				outputErrorJSON(cmd, err)
				return
			}
			_ = outputJSON(cmd.OutOrStdout(), map[string]interface{}{"ok": true, "type": "watched", "code": code})
		default:
			outputErrorJSON(cmd, fmt.Errorf("remove only supports -t want_watch or -t watched"))
		}
	},
}

func init() {
	rootCmd.AddCommand(bookmarksCmd)
	bookmarksCmd.Flags().StringVarP(&bookmarkType, "type", "t", "want_watch", "Type: want_watch, watched, actors")
	bookmarksCmd.Flags().IntVarP(&bookmarkPage, "page", "p", 1, "Page")

	bookmarksCmd.AddCommand(bookmarksAddCmd)
	bookmarksAddCmd.Flags().StringVarP(&bookmarkAddRemoveType, "type", "t", "want_watch", "Type: want_watch, watched")
	bookmarksAddCmd.Flags().IntVarP(&bookmarkAddRating, "rating", "r", 0, "Rating 1-5 for -t watched")
	bookmarksAddCmd.Flags().StringVarP(&bookmarkAddContent, "content", "c", "", "Comment for -t watched (10-1000 chars)")

	bookmarksCmd.AddCommand(bookmarksRemoveCmd)
	bookmarksRemoveCmd.Flags().StringVarP(&bookmarkAddRemoveType, "type", "t", "want_watch", "Type: want_watch, watched")
}
