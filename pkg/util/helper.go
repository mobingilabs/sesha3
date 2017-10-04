package util

import "github.com/spf13/cobra"

func flag(cmd *cobra.Command, f string) string {
	s := cmd.Flag(f).DefValue
	if cmd.Flag(f).Changed {
		s = cmd.Flag(f).Value.String()
	}

	return s
}

func GetCliStringFlag(cmd *cobra.Command, f string) string {
	return flag(cmd, f)
}
