package cmd

import (
	"os"

	"github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/sesha3/pkg/cert"
	"github.com/spf13/cobra"
)

func SetupRoute53Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "r53",
		Short: "setup route53",
		Long:  `Add local url to route53.`,
		Run: func(cmd *cobra.Command, args []string) {
			d, err := cert.AddLocalUrlToRoute53()
			if err != nil {
				debug.Error(err)
				os.Exit(1)
			}

			debug.Info("added to route53:", d)
		},
	}

	cmd.Flags().SortFlags = false
	return cmd
}
