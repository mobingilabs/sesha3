package main

import (
	"fmt"
	"log"
	"os"

	"github.com/mobingilabs/settyd/awsports"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sesha3",
	Short: "Secure Shell and Application Access Server",
	Long:  "Mobingi Secure Shell and Application Access Server.",
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("hello")
		_, err := os.Stat("./certs/")

		if err == nil {
			fmt.Println("./certs detected.")
		} else {
			fmt.Println("./certs not detected. mkdir ./certs")
			os.Mkdir("./certs", 0700)
		}

		err = awsports.Download(awsRegion, profilename)
		fmt.Println(err)
		signalHandler()
		serve()
	},
}

func init() {
	rootCmd.Flags().SortFlags = false
	rootCmd.PersistentFlags().String("env", "dev", "dev, test, prod")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}
