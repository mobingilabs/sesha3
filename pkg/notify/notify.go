package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	as "github.com/aws/aws-sdk-go/aws/session"
	"github.com/guregu/dynamo"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/pkg/errors"
)

type EventN struct {
	Server_name string `dynamo:"server_name"`
	Slack       string `dynamo:"slack"`
}

type Notificate struct {
	Slack  bool
	Cred   string
	Region string
	URLs   EventN
	Valid  bool
}

func (n *Notificate) Dynamoget() (EventN, error) {
	serverName := "sesha3"
	var results []EventN
	cred := credentials.NewSharedCredentials("/root/.aws/credentials", n.Cred)
	db := dynamo.New(as.New(), &aws.Config{Region: aws.String(n.Region),
		Credentials: cred,
	})
	table := db.Table("SESHA3")
	err := table.Get("server_name", serverName).All(&results)
	url := results[0]
	return url, err
}

func (w *Notificate) WebhookNotification(v interface{}) error {
	if w.Valid != true {
		return errors.New("invalid")
	}

	type payload_t struct {
		Text string `json:"text"`
	}

	var urls []string
	if w.Slack {
		NotificateURL := w.URLs
		urls = append(urls, NotificateURL.Slack)
	}

	var err_string string
	err_string = time.Now().Format(time.RFC1123) + "\n"

	switch v.(type) {
	case string:
		err := v.(string)
		err_string += "info: " + fmt.Sprintf("%v", err)
	case error:
		err_string += "error: " + fmt.Sprintf("%+v", errors.WithStack(v.(error)))
	default:
		err_string += fmt.Sprintf("%s", v)
	}

	err_string = "```" + err_string + "```"
	payload := payload_t{
		Text: err_string,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "payload marshal failed")
	}

	client := &http.Client{}
	for _, ep := range urls {
		req, err := http.NewRequest(http.MethodPost, ep, bytes.NewBuffer(b))
		req.Header.Add("Content-Type", "application/json")
		_, err = client.Do(req)
		if err != nil {
			return errors.Wrap(err, "notification client do failed")
		}
	}

	return err
}

var Notifier Notificate

func errcheck(v interface{}) {
	var err error
	switch v.(type) {
	case string:
		str := v.(string)
		if str != "" {
			err = Notifier.WebhookNotification(str)
		}
	case error:
		terr := v.(error)
		if terr != nil {
			err = Notifier.WebhookNotification(terr.Error())
		}
	default:
		str := fmt.Sprintf("%v", v)
		if str != "" {
			err = Notifier.WebhookNotification(str)
		}
	}

	if err != nil {
		debug.Error(errors.Wrap(err, "webhook notify failed"))
	}
}

func HookPost(v interface{}) {
	switch v.(type) {
	case string:
		err := v.(string)
		go errcheck(err)
	case error:
		err := v.(error)
		go errcheck(err)
	default:
		err := fmt.Sprintf("%v", v)
		go errcheck(err)
	}
}
