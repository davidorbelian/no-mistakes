package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Report that this build never updates itself",
		Args:  cobra.ArbitraryArgs,
		// The upstream flags (--beta, -y, --force) are gone, but any of them
		// must still land on the refusal below instead of a flag-parse error,
		// so every habit and script gets the reason.
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("self-update is disabled in this build: it never downloads or installs releases; update it where it is built and installed from source")
		},
	}
}
