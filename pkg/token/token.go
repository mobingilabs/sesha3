package token

import (
	"github.com/aws/aws-sdk-go/aws"
	as "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/golang/glog"
	"github.com/guregu/dynamo"
	"github.com/mobingilabs/sesha3/pkg/util"
)

type event struct {
	User   string `dynamo:"username"`
	Pass   string `dynamo:"password"`
	Status string `dynamo:"status"`
}

type root struct {
	ApiToken string `json:"api_token"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Status   string `json:"status"`
}

func CheckToken(uname string, pwdmd5 string) (bool, error) {
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
		if pwdmd5 == data.Pass && data.Status != "deleted" {
			glog.Infof("valid subuser: %v", uname)
			return true, nil
		}
	}

	if err != nil {
		glog.Errorf("error in table get: %v", err)
	}

	// try looking at the root users table
	var queryInput = &dynamodb.QueryInput{
		TableName:              aws.String("MC_USERS"),
		IndexName:              aws.String("email-index"),
		KeyConditionExpression: aws.String("email = :e"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":e": {
				S: aws.String(uname),
			},
		},
	}

	dbsvc := dynamodb.New(sess, &aws.Config{
		Region: aws.String(util.GetRegion()),
	})

	resp, err := dbsvc.Query(queryInput)
	if err != nil {
		glog.Errorf("query failed: %v", err)
	} else {
		ru := []root{}
		err = dynamodbattribute.UnmarshalListOfMaps(resp.Items, &ru)
		if err != nil {
			glog.Errorf("dynamo obj unmarshal failed: %v", err)
		}

		glog.Infof("raw: %v", ru)

		// should be a valid root user
		for _, u := range ru {
			if u.Email == uname && u.Password == pwdmd5 {
				if u.Status == "" || u.Status == "trial" {
					glog.Infof("valid root user: %v", uname)
					ret = true
					break
				}
			}
		}
	}

	return ret, err
}
