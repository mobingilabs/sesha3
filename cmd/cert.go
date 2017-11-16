package cmd

import (
	"strings"

	"github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/sesha3/pkg/params"
	"github.com/mobingilabs/sesha3/pkg/util"
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
			iid := strings.Replace(util.GetEc2Id(), "-", "", -1)
			domain := "sesha3-" + iid + ".labs.mobingi.com"
			if params.IsDev {
				domain = "sesha3-" + iid + ".mobingi.com"
			}

			debug.Info("domain:", domain)
		},
	}

	cmd.Flags().SortFlags = false
	return cmd
}
