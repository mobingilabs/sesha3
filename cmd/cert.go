package cmd

import (
	"os"

	"github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/sesha3/pkg/cert"
	"github.com/spf13/cobra"
)

// SetupHttpsCmd attempts to install LetsEncrypt certificates locally using the instance id
// and registers it to Route53. This function assumes that certbot is already installed.
func SetupHttpsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cert",
		Short: "setup https",
		Long:  `Setup https support.`,
		Run: func(cmd *cobra.Command, args []string) {
			err := cert.SetupLetsEncryptCert(true)
			if err != nil {
				debug.Error(err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().SortFlags = false
	return cmd
}
