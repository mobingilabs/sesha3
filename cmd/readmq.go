package cmd

import (
	"os"
	"strconv"

	"github.com/flowerinthenight/rmq"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

// SetupHttpsCmd attempts to install LetsEncrypt certificates locally using the instance id
// and registers it to Route53. This function assumes that certbot is already installed.
func SetupReadMqCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "readmq",
		Short: "read mochi message queue for requests",
		Long:  `Read mochi message queue for requests.`,
		Run: func(cmd *cobra.Command, args []string) {
			port, err := strconv.Atoi(os.Getenv("RABBITMQ_PORT"))
			if err != nil {
				glog.Fatalf("port env failed: %v", err)
			}

			con := rmq.New(&rmq.Config{
				Host:     os.Getenv("RABBITMQ_HOST"),
				Port:     port,
				Username: os.Getenv("RABBITMQ_USER"),
				Password: os.Getenv("RABBITMQ_PASS"),
				Vhost:    "/",
			})

			err = con.Connect()
			if err != nil {
				glog.Fatalf("connect failed: %v", err)
			}

			defer con.Close()

			bindId, err := con.AddBinding(&rmq.BindConfig{
				ExchangeOpt: &rmq.ExchangeOptions{
					Name:       "sesha3.exchange.direct",
					Type:       "direct",
					Durable:    false,
					AutoDelete: true,
				},
				QueueOpt: &rmq.QueueOptions{
					QueueName:  "sesha3.queue.ttyurl",
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

			glog.Infof("binding added (id = %v)", bindId)
			if err != nil {
				glog.Fatalf("add binding failed: %v", err)
			}

			forever := make(chan int)
			<-forever
		},
	}

	cmd.Flags().SortFlags = false
	return cmd
}

func ProcessMessage(payload []byte) error {
	glog.Infof("payload: %s", payload)
	return nil
}
