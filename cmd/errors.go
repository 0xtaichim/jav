package cmd

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/spf13/cobra"
	"github.com/taichi/javcli/pkg/javdb"
)

// outputErrorJSON writes a JSON error object to the command's stdout.
// For login-required errors: {"code": 401, "message": "xxx"}.
// Otherwise: {"error": "<message>"}.
func outputErrorJSON(c *cobra.Command, err error) {
	out := c.OutOrStdout()
	if out == nil {
		return
	}
	enc := json.NewEncoder(out)
	var lr *javdb.LoginRequiredError
	if errors.As(err, &lr) {
		msg := lr.Message
		if msg == "" {
			msg = "Unauthorized"
		}
		_ = enc.Encode(map[string]interface{}{"code": 401, "message": msg})
		return
	}
	_ = enc.Encode(map[string]string{"error": err.Error()})
}

// outputJSON writes v as JSON to w.
func outputJSON(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	return enc.Encode(v)
}
