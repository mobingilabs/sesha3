package cmd

import (
	goflag "flag"

	"github.com/mobingilabs/mobingi-sdk-go/pkg/cmdline"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/sesha3/pkg/params"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

var rootCmd = &cobra.Command{
	Use:   "sesha3",
	Short: "Secure Shell and Application Access Server",
	Long:  "Mobingi Secure Shell and Application Access Server.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		goflag.Parse()
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		debug.ErrorTraceExit(err, 1)
	}
}

func init() {
	rootCmd.Flags().SortFlags = false
	rootCmd.PersistentFlags().BoolVar(&params.IsDev, "rundev", params.IsDev, "run as dev, otherwise, prod")
	rootCmd.PersistentFlags().StringArray("notify-endpoints", []string{"slack"}, "values: slack")
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
		SetupRoute53Cmd(),
		SetupHttpsCmd(),
		SetupReadMqCmd(),
	)

	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
}
