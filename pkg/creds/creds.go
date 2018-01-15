package creds

import (
	"crypto/md5"
	"fmt"

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
	var md5p string

	if c.GrantType == "hashpassword" {
		glog.V(1).Infof("input password already hashed, use directly")
		md5p = c.Password
	} else {
		glog.V(1).Infof("will compute md5.sum() to password")
		md5p = fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s", c.Password))))

		// test
		sh, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
		if err != nil {
			glog.Errorf("hashfrompass failed: %+v", err)
		}

		glog.V(2).Infof("test:hashed: %v, %v", sh, string(sh))
	}

	glog.V(2).Infof("username: %v", c.Username)
	glog.V(2).Infof("hashed passwd: %v", md5p)

	// look in subusers first
	table := db.Table("MC_IDENTITY")
	err := table.Get("username", c.Username).All(&results)
	for _, data := range results {
		glog.V(2).Infof("data: %+v", data)

		// test
		err = bcrypt.CompareHashAndPassword([]byte(data.Pass), []byte(c.Password))
		if err != nil {
			glog.Errorf("test:invalidpass: %+v", err)
		} else {
			glog.V(2).Infof("valid password")
		}

		if md5p == data.Pass && data.Status != "deleted" {
			glog.V(1).Infof("valid subuser: %v", c.Username)
			return true, nil
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

		glog.V(2).Infof("raw: %v", ru)

		// should be a valid root user
		for _, u := range ru {
			if u.Email == c.Username && u.Password == md5p {
				if u.Status == "" || u.Status == "trial" {
					glog.V(1).Infof("valid root user: %v", c.Username)
					ret = true
					break
				}
			}
		}
	}

	return ret, err
}
