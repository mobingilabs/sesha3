package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	as "github.com/aws/aws-sdk-go/aws/session"
	"github.com/guregu/dynamo"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
)

type event struct {
	Username string `dynamo:"username"`
	Pass     string `dynamo:"password"`
	Status   string `dynamo:"status"`
}

func checkToken(credential string, region string, token_user string, token_pass string) (bool, error) {
	cred := credentials.NewSharedCredentials("/root/.aws/credentials", credential)
	db := dynamo.New(as.New(), &aws.Config{Region: aws.String(region),
		Credentials: cred,
	})

	table := db.Table("MC_IDENTITY")
	var results []event
	err := table.Get("username", token_user).All(&results)
	ret := false
	for _, data := range results {
		if data.Status == "deleted" {
			ret = false
			d.Info("token_ALMuser_check: status=deleted, username:", data.Username)
			break
		} else {
			d.Info("token_ALMuser_check: status=OK, username:", data.Username)
		}

		if token_pass == data.Pass {
			d.Info("token_ALMuser_check: success")
			ret = true
		}
	}

	return ret, err
}
