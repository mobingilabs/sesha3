package token

import (
	"github.com/aws/aws-sdk-go/aws"
	as "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/guregu/dynamo"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/sesha3/pkg/util"
	"github.com/pkg/errors"
)

type event struct {
	User   string `dynamo:"username"`
	Pass   string `dynamo:"password"`
	Status string `dynamo:"status"`
}

type root struct {
	Email  string `dynamo:"email"`
	Pass   string `dynamo:"password"`
	Status string `dynamo:"status"`
}

func CheckToken(uname string, pwd string) (bool, error) {
	sess := as.Must(as.NewSessionWithOptions(as.Options{
		SharedConfigState: as.SharedConfigDisable,
	}))

	db := dynamo.New(sess, &aws.Config{
		Region: aws.String(util.GetRegion()),
	})

	var results []event
	var ret bool

	// look in subusers first
	table := db.Table("MC_IDENTITY")
	err := table.Get("username", uname).All(&results)
	for _, data := range results {
		if pwd == data.Pass && data.Status != "deleted" {
			d.Info("valid subuser:", uname)
			return true, nil
		}
	}

	if err != nil {
		d.Error("error in table get:", err)
	}

	/*
		var results []event
		ret := false

		// look in subusers first
		table := db.Table("MC_IDENTITY")
		err := table.Get("username", uname).All(&results)
		for _, data := range results {
			if data.Status == "deleted" {
				ret = false
				d.Info("token_ALMuser_check: status=deleted, username:", data.User)
				break
			} else {
				d.Info("token_ALMuser_check: status=OK, username:", data.User)
			}

			if pwd == data.Pass {
				d.Info("token_ALMuser_check: success")
				ret = true
			}
		}
	*/

	// try looking at the root users table
	var queryInput = &dynamodb.QueryInput{
		TableName: aws.String("MC_USERS"),
		IndexName: aws.String("email-index"),
		KeyConditions: map[string]*dynamodb.Condition{
			"modifier": {
				ComparisonOperator: aws.String("EQ"),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(uname),
					},
				},
			},
		},
	}

	dbsvc := dynamodb.New(sess, &aws.Config{
		Region: aws.String(util.GetRegion()),
	})

	resp, err := dbsvc.Query(queryInput)
	if err != nil {
		d.Error(err)
	} else {
		ru := []root{}
		err = dynamodbattribute.UnmarshalListOfMaps(resp.Items, &ru)
		if err != nil {
			d.Error(errors.Wrap(err, "dynamo obj unmarshal failed"))
		}

		d.Info("raw:", ru)

		// should be a valid root user
		for _, u := range ru {
			if u.Email == uname && u.Pass == pwd {
				if u.Status == "" || u.Status == "trial" {
					d.Info("valid root user:", uname)
					ret = true
					break
				}
			}
		}
	}

	// return false, errors.New("user not found")
	return ret, err
}
