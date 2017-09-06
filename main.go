package main

import (
	"log"
	"log/syslog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mobingilabs/sesha3/awsports"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sesha3",
	Short: "Secure Shell and Application Access Server",
	Long:  "Mobingi Secure Shell and Application Access Server.",
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("hello!")
		_, err := os.Stat("./certs/")

		if err == nil {
			log.Println("./certs detected.")
		} else {
			log.Println("./certs not detected. mkdir ./certs")
			os.Mkdir("./certs", 0700)
		}

		env := GetCliStringFlag(cmd, "env")
		syslogging = GetCliStringFlag(cmd, "syslog")
		if syslogging == "enable" {
			logger, err = syslog.New(syslog.LOG_NOTICE|syslog.LOG_USER, "sesha3")
			if err != nil {
				panic(err)
			}
			log.SetOutput(logger)
		}

		domain = GetCliStringFlag(cmd, "domain")
		region = GetCliStringFlag(cmd, "aws-region")
		ec2id = GetCliStringFlag(cmd, "ec2-id")
		credprof = GetCliStringFlag(cmd, "cred-profile")
		err = awsports.Download(env, region, credprof)
		log.Println(err)
		serve(cmd)
	},
}

func init() {
	rootCmd.Flags().SortFlags = false
	rootCmd.PersistentFlags().String("env", "dev", "values: dev, test, prod")
	rootCmd.PersistentFlags().String("syslog", "disable", "values: enable, disable")
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
