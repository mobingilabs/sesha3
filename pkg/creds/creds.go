package creds

import (
	"github.com/aws/aws-sdk-go/aws"
	as "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/golang/glog"
	"github.com/guregu/dynamo"
	"github.com/mobingilabs/sesha3/pkg/util"
	"golang.org/x/crypto/bcrypt"
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

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"passwd"`

	// GrantType is the type of password this object has.
	// Valid value(s): 'hashpassword'
	GrantType string `json:"grant_type,omitempty"`
}

func (c *Credentials) Validate() (bool, error) {
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
	err := table.Get("username", c.Username).All(&results)
	for _, data := range results {
		glog.V(2).Infof("data: %+v", data)
		if c.GrantType == "hashpassword" {
			// direct text compare here
			if data.Pass == c.Password && data.Status != "deleted" {
				glog.V(1).Infof("(direct text compare) valid subuser: %v", c.Username)
				return true, nil
			}
		} else {
			err = bcrypt.CompareHashAndPassword([]byte(data.Pass), []byte(c.Password))
			if err == nil {
				glog.V(1).Infof("(blowfish) valid subuser: %v", c.Username)
				return true, nil
			}
		}
	}

	if err != nil {
		glog.Errorf("error in table get: %+v", util.ErrV(err))
	}

	// try looking at the root users table
	var queryInput = &dynamodb.QueryInput{
		TableName:              aws.String("MC_USERS"),
		IndexName:              aws.String("email-index"),
		KeyConditionExpression: aws.String("email = :e"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":e": {
				S: aws.String(c.Username),
			},
		},
	}

	dbsvc := dynamodb.New(sess, &aws.Config{
		Region: aws.String(util.GetRegion()),
	})

	resp, err := dbsvc.Query(queryInput)
	if err != nil {
		glog.Errorf("query failed: %+v", util.ErrV(err))
	} else {
		ru := []root{}
		err = dynamodbattribute.UnmarshalListOfMaps(resp.Items, &ru)
		if err != nil {
			glog.Errorf("dynamo obj unmarshal failed: %+v", util.ErrV(err))
		}

		glog.V(2).Infof("raw root user: %+v", ru)

		// should be a valid root user
		for _, u := range ru {
			glog.V(2).Infof("raw root user (iter): %+v", u)
			if c.GrantType == "hashpassword" {
				if u.Email == c.Username && u.Password == c.Password {
					if u.Status == "" || u.Status == "trial" {
						glog.V(1).Infof("valid root user: %v", c.Username)
						ret = true
						break
					}
				}
			} else {
				if u.Status == "" || u.Status == "trial" {
					err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(c.Password))
					if err == nil {
						glog.V(1).Infof("(blowfish) valid root user: %v", c.Username)
						return true, nil
					}
				}
			}
		}
	}

	return ret, err
}
