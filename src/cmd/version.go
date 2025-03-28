package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const versionDesc = "Displays the version"

// The current Git Town version (set at compile time).
var version string

// The time this Git Town binary was compiled (set at compile time).
var buildDate string //nolint:gochecknoglobals

// versionCmd represents the version command.
func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "version",
		GroupID: "setup",
		Args:    cobra.NoArgs,
		Short:   versionDesc,
		Long:    long(versionDesc),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Git Town %s (%s)\n", version, buildDate)
		},
	}
}
