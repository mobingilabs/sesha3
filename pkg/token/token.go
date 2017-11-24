package token

import (
	"github.com/aws/aws-sdk-go/aws"
	as "github.com/aws/aws-sdk-go/aws/session"
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

func CheckToken(uname string, pwd string) (bool, error) {
	sess := as.Must(as.NewSessionWithOptions(as.Options{
		SharedConfigState: as.SharedConfigDisable,
	}))

	db := dynamo.New(sess, &aws.Config{
		Region: aws.String(util.GetRegion()),
	})

	var results []event

	// look in subusers first
	table := db.Table("MC_IDENTITY")
	err := table.Get("username", uname).All(&results)
	if err != nil {
		for _, data := range results {
			if pwd == data.Pass && data.Status != "deleted" {
				d.Info("valid subuser:", uname)
				return true, nil
			}
		}
	}

	/*
		var queryInput = &dynamodb.QueryInput{
			TableName: aws.String("person"),
			IndexName: aws.String("firstName-index"),
			KeyConditions: map[string]*dynamodb.Condition{
				"modifier": {
					ComparisonOperator: aws.String("EQ"),
					AttributeValueList: []*dynamodb.AttributeValue{
						{
							S: aws.String("David"),
						},
					},
				},
			},
		}

		var resp1, err1 = svc.Query(queryInput)
		if err1 != nil {
			fmt.Println(err1)
		} else {
			personObj := []Person{}
			err = dynamodbattribute.UnmarshalListOfMaps(resp1.Items, &personObj)
			log.Println(personObj)
		}
	*/

	return false, errors.New("user not found")
}
