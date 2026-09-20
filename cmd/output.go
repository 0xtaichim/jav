package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/spf13/cobra"
	"github.com/taichi/javcli/pkg/javdb"
)

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeErrorJSON(w io.Writer, err error) error {
	enc := json.NewEncoder(w)
	var lr *javdb.LoginRequiredError
	if errors.As(err, &lr) {
		msg := lr.Message
		if msg == "" {
			msg = "Unauthorized"
		}
		return enc.Encode(map[string]any{"code": 401, "message": msg})
	}
	return enc.Encode(map[string]string{"error": err.Error()})
}

func withClient(cmd *cobra.Command, fn func(context.Context, *javdb.Client) (any, error)) error {
	v, err := fn(cmd.Context(), clientFrom(cmd))
	if err != nil {
		return err
	}
	return writeJSON(cmd.OutOrStdout(), v)
}
