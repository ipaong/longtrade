package onboard

import (
	"embed"

	"github.com/spf13/cobra"
)

//go:generate cp -r ../../../../workspace .
//go:embed workspace
var embeddedFiles embed.FS

func NewOnboardCommand() *cobra.Command {
	var nonInteractive bool

	cmd := &cobra.Command{
		Use:     "onboard",
		Aliases: []string{"o"},
		Short:   "Initialize khunquant configuration and workspace",
		Run: func(cmd *cobra.Command, args []string) {
			onboard(nonInteractive)
		},
	}

	cmd.Flags().BoolVarP(&nonInteractive, "yes", "y", false,
		"Non-interactive setup: accept the terms, skip credential encryption, and skip portfolio prompts (used by the launcher)")

	return cmd
}
