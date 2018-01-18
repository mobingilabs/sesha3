package cmd

import (
	"encoding/json"
	"io/ioutil"
	"strconv"

	"github.com/flowerinthenight/rmq"
	"github.com/golang/glog"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/private"
	"github.com/spf13/cobra"
)

type rmqConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"user"`
	Password string `json:"pass"`
}

var confFile string

// SetupHttpsCmd attempts to install LetsEncrypt certificates locally using the instance id
// and registers it to Route53. This function assumes that certbot is already installed.
func SetupReadMqCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "readmq",
		Short: "read mochi message queue for requests",
		Long:  `Read mochi message queue for requests.`,
		Run: func(cmd *cobra.Command, args []string) {
			if !private.Exists(confFile) {
				glog.Fatalf("cannot find rmq.conf")
			}

			b, err := ioutil.ReadFile(confFile)
			if err != nil {
				glog.Fatalf("read rmq.conf failed: %v", err)
			}

			var conf rmqConfig
			err = json.Unmarshal(b, &conf)
			if err != nil {
				glog.Fatalf("unmarshal rmq.conf failed: %v", err)
			}

			port, err := strconv.Atoi(conf.Port)
			if err != nil {
				glog.Fatalf("port failed: %v", err)
			}

			con := rmq.New(&rmq.Config{
				Host:     conf.Host,
				Port:     port,
				Username: conf.Username,
				Password: conf.Password,
				Vhost:    "/",
			})

			err = con.Connect()
			if err != nil {
				glog.Fatalf("connect failed: %v", err)
			}

			defer con.Close()

			bindId, err := con.AddBinding(&rmq.BindConfig{
				ExchangeOpt: &rmq.ExchangeOptions{
					Name:       "sesha3.direct",
					Type:       "direct",
					Durable:    false,
					AutoDelete: true,
				},
				QueueOpt: &rmq.QueueOptions{
					QueueName:  "sesha3.cmd",
					Durable:    false,
					AutoDelete: true,
				},
				QueueBindOpt: &rmq.QueueBindOptions{
					RoutingKey: "rk-sesha3",
				},
				ConsumeOpt: &rmq.ConsumeOptions{
					ClientTag:  "sesha3dispatchertag",
					FnCallback: ProcessMessage,
				},
			})

			if err != nil {
				glog.Fatalf("add binding failed: %v", err)
			}

			glog.Infof("binding added (id = %v)", bindId)

			forever := make(chan int)
			<-forever
		},
	}

	cmd.Flags().SortFlags = false
	cmd.Flags().StringVar(&confFile, "rmq-conf", "rmq.conf", "rmq config file")
	return cmd
}

func ProcessMessage(payload []byte) error {
	glog.Infof("payload: %s", payload)
	return nil
}
