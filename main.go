package main

import (
	"log"
	"log/syslog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/sesha3/awsports"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sesha3",
	Short: "Secure Shell and Application Access Server",
	Long:  "Mobingi Secure Shell and Application Access Server.",
	Run: func(cmd *cobra.Command, args []string) {
		if syslogging {
			logger, err := syslog.New(syslog.LOG_NOTICE|syslog.LOG_USER, "sesha3")
			d.ErrorExit(err, 1)

			log.SetFlags(0)
			log.SetOutput(logger)
		}

		name, err := os.Executable()
		sourcedir := filepath.Dir(name)
		env := GetCliStringFlag(cmd, "env")
		_, err = os.Stat(sourcedir + "/certs/")
		if err == nil {
			log.Println(sourcedir + "/certs detected.")
		} else {
			log.Println(sourcedir + "/certs not detected. mkdir certs dir")
			err = os.MkdirAll(sourcedir+"/certs", 0700)
			if err != nil {
				log.Println("error:", errors.Wrap(err, "mkdirall failed"))
			}
		}

		domain = GetCliStringFlag(cmd, "domain")
		region = GetCliStringFlag(cmd, "aws-region")
		ec2id = GetCliStringFlag(cmd, "ec2-id")
		credprof = GetCliStringFlag(cmd, "cred-profile")
		err = awsports.Download(env, region, credprof)
		log.Println(err)
		serve(cmd)
		d.Info()
	},
}

func init() {
	rootCmd.Flags().SortFlags = false
	rootCmd.PersistentFlags().String("env", "dev", "values: dev, test, prod")
	rootCmd.PersistentFlags().BoolVar(&syslogging, "syslog", false, "set log output to syslog")
	rootCmd.PersistentFlags().String("domain", "sesha3.labs.mobingi.com", "server domain")
	rootCmd.PersistentFlags().String("port", "80", "server port")
	rootCmd.PersistentFlags().String("aws-region", "ap-northeast-1", "aws region")
	rootCmd.PersistentFlags().String("ec2-id", "i-0d6ff50d6caef8ffa", "ec2 server instance id")
	rootCmd.PersistentFlags().String("cred-profile", "sesha3", "aws credenfile profile name")
}

func signalHandler() {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(
		sigchan,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	go func() {
		for {
			s := <-sigchan
			switch s {
			case syscall.SIGINT, syscall.SIGTERM:
				os.Exit(0)
			}
		}
	}()
}

func main() {
	signalHandler()
	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}
