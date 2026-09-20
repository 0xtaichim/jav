package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/taichi/javcli/pkg/javdb"
)

var bookmarksCmd = &cobra.Command{
	Use:   "bookmarks",
	Short: "List bookmarks",
	Long:  `List want_watch, watched, or actors. Requires JAVDB_COOKIES. Output is JSON.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		typ, err := cmd.Flags().GetString("type")
		if err != nil {
			return err
		}
		page, err := cmd.Flags().GetInt("page")
		if err != nil {
			return err
		}
		return withClient(cmd, func(ctx context.Context, c *javdb.Client) (any, error) {
			return c.GetBookmarks(ctx, typ, page)
		})
	},
}

var bookmarksAddCmd = &cobra.Command{
	Use:   "add [code]",
	Short: "Add bookmark",
	Long:  `Add want_watch: bookmarks add -t want_watch <code>. Add watched: bookmarks add -t watched <code> [--rating 1-5] [--content "comment"]`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		typ, err := cmd.Flags().GetString("type")
		if err != nil {
			return err
		}
		rating, err := cmd.Flags().GetInt("rating")
		if err != nil {
			return err
		}
		content, err := cmd.Flags().GetString("content")
		if err != nil {
			return err
		}
		code := args[0]
		return withClient(cmd, func(ctx context.Context, c *javdb.Client) (any, error) {
			switch typ {
			case "want_watch":
				if err := c.AddWantWatch(ctx, code); err != nil {
					return nil, err
				}
			case "watched":
				if err := c.AddWatched(ctx, code, rating, content); err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("add only supports -t want_watch or -t watched")
			}
			return map[string]any{"ok": true, "type": typ, "code": code}, nil
		})
	},
}

var bookmarksRemoveCmd = &cobra.Command{
	Use:   "remove [code]",
	Short: "Remove bookmark",
	Long:  `Remove want_watch: bookmarks remove -t want_watch <code>. Remove watched: bookmarks remove -t watched <code>`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		typ, err := cmd.Flags().GetString("type")
		if err != nil {
			return err
		}
		code := args[0]
		return withClient(cmd, func(ctx context.Context, c *javdb.Client) (any, error) {
			switch typ {
			case "want_watch":
				if err := c.RemoveWantWatch(ctx, code); err != nil {
					return nil, err
				}
			case "watched":
				if err := c.RemoveWatched(ctx, code); err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("remove only supports -t want_watch or -t watched")
			}
			return map[string]any{"ok": true, "type": typ, "code": code}, nil
		})
	},
}

func init() {
	rootCmd.AddCommand(bookmarksCmd)
	bookmarksCmd.Flags().StringP("type", "t", "want_watch", "Type: want_watch, watched, actors")
	bookmarksCmd.Flags().IntP("page", "p", 1, "Page")

	bookmarksCmd.AddCommand(bookmarksAddCmd)
	bookmarksAddCmd.Flags().StringP("type", "t", "want_watch", "Type: want_watch, watched")
	bookmarksAddCmd.Flags().IntP("rating", "r", 0, "Rating 1-5 for -t watched")
	bookmarksAddCmd.Flags().StringP("content", "c", "", "Comment for -t watched (10-1000 chars)")

	bookmarksCmd.AddCommand(bookmarksRemoveCmd)
	bookmarksRemoveCmd.Flags().StringP("type", "t", "want_watch", "Type: want_watch, watched")
}
