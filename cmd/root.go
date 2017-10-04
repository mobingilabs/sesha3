package cmd

import (
	"github.com/mobingilabs/mobingi-sdk-go/pkg/cmdline"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/sesha3/pkg/params"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sesha3",
	Short: "Secure Shell and Application Access Server",
	Long:  "Mobingi Secure Shell and Application Access Server.",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		debug.ErrorTraceExit(err, 1)
	}
}

func init() {
	rootCmd.Flags().SortFlags = false
	rootCmd.PersistentFlags().StringVar(&params.Environment, "env", "dev", "values: dev, test, prod")
	rootCmd.PersistentFlags().BoolVar(&params.UseSyslog, "syslog", false, "set log output to syslog")
	rootCmd.PersistentFlags().StringVar(&params.Domain, "domain", "sesha3.labs.mobingi.com", "server domain")
	rootCmd.PersistentFlags().String("port", "443", "server port")
	rootCmd.PersistentFlags().StringArray("notify-endpoints", []string{"slack"}, "values: slack")
	rootCmd.PersistentFlags().StringVar(&params.Region, "aws-region", "ap-northeast-1", "aws region")
	rootCmd.PersistentFlags().StringVar(&params.Ec2Id, "ec2-id", "i-0d6ff50d6caef8ffa", "ec2 server instance id")
	rootCmd.PersistentFlags().StringVar(&params.CredProfile, "cred-profile", "sesha3", "aws credenfile profile name")
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:   "help",
		Short: "help about any command",
		Long: `Help provides help for any command in the application.
Simply type '` + cmdline.Args0() + ` help [path to command]' for full details.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	})

	// add supported cmds
	rootCmd.AddCommand(
		ServeCmd(),
	)
}
