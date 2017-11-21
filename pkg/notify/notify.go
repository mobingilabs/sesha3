package notify

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	as "github.com/aws/aws-sdk-go/aws/session"
	"github.com/guregu/dynamo"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/notification"
	"github.com/mobingilabs/sesha3/pkg/params"
	"github.com/pkg/errors"
)

type EventN struct {
	ServerName string `dynamo:"server_name"`
	SlackUrl   string `dynamo:"slack"`
}

type HttpNotifier struct {
	region    string
	credprof  string
	notifiers []notification.Notifier
	valid     bool
}

func (n *HttpNotifier) Init(eps []string) error {
	n.region = params.Region
	n.credprof = params.CredProfile
	n.notifiers = make([]notification.Notifier, 0)

	// iterate endpoints
	for _, ep := range eps {
		switch ep {
		case "slack":
			su, err := n.getSlackUrl()
			if err != nil {
				debug.Error(err)
				return err
			}

			hn := notification.NewSimpleHttpNotify(su.SlackUrl)
			n.notifiers = append(n.notifiers, hn)
		default:
			hn := notification.NewSimpleHttpNotify(ep)
			n.notifiers = append(n.notifiers, hn)
		}
	}

	if len(n.notifiers) == 0 {
		return errors.New("no notify endpoints")
	}

	n.valid = true
	return nil
}

func (n *HttpNotifier) getSlackUrl() (EventN, error) {
	serverName := "sesha3"
	var results []EventN
	// cred := credentials.NewSharedCredentials("/root/.aws/credentials", n.credprof)
	sess := as.Must(as.NewSessionWithOptions(as.Options{
		SharedConfigState: as.SharedConfigDisable,
	}))

	db := dynamo.New(sess, &aws.Config{
		Region: aws.String(n.region),
		// Credentials: cred,
	})

	table := db.Table("SESHA3")
	err := table.Get("server_name", serverName).All(&results)
	url := results[0]
	return url, err
}

func (n *HttpNotifier) Notify(v interface{}) error {
	if !n.valid {
		return errors.New("not properly initialized")
	}

	type payload_t struct {
		Text string `json:"text"`
	}

	var str string
	str = time.Now().Format(time.RFC1123) + "\n"

	switch v.(type) {
	case string:
		err := v.(string)
		str += fmt.Sprintf("%v", err)
	case error:
		str += "error: " + fmt.Sprintf("%+v", errors.WithStack(v.(error)))
	default:
		str += fmt.Sprintf("%s", v)
	}

	str = "```" + str + "```"
	payload := payload_t{
		Text: str,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "payload marshal failed")
	}

	for _, sender := range n.notifiers {
		go func() {
			err := sender.Notify(b)
			if err != nil {
				debug.Error(errors.Wrap(err, "notify failed"))
			}
		}()
	}

	return nil
}

var Notifier HttpNotifier

func HookPost(v interface{}) {
	var err error
	switch v.(type) {
	case string:
		str := v.(string)
		if str != "" {
			go func() {
				err = Notifier.Notify(str)
			}()
		}
	case error:
		terr := v.(error)
		if terr != nil {
			go func() {
				err = Notifier.Notify(terr)
			}()
		}
	default:
		// don't bother
	}

	if err != nil {
		debug.Error(errors.Wrap(err, "webhook notify failed"))
	}
}
