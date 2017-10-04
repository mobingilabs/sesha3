package awsports

import (
	"math/rand"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/cmdline"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/private"
	"github.com/pkg/errors"
)

type SecurityGroupRequest struct {
	AuthorizeSecurityGroupIngressInput *ec2.AuthorizeSecurityGroupIngressInput
	RevokeSecurityGroupIngressInput    *ec2.RevokeSecurityGroupIngressInput
	DescribeSecurityGroupsInput        *ec2.DescribeSecurityGroupsInput
	Sess                               *session.Session
	Cred                               *credentials.Credentials
	InstanceID                         string
	OpenPortList                       []int64
	SecurityID                         string
	RequestPort                        int64
	Ec2client                          *ec2.EC2
}

func (s *SecurityGroupRequest) CreateRequest() {
	//goroupid
	s.DescribeSecurityGroupsInput = &ec2.DescribeSecurityGroupsInput{GroupIds: []*string{aws.String(s.SecurityID)}}
	s.AuthorizeSecurityGroupIngressInput = &ec2.AuthorizeSecurityGroupIngressInput{GroupId: aws.String(s.SecurityID)}
	s.RevokeSecurityGroupIngressInput = &ec2.RevokeSecurityGroupIngressInput{GroupId: aws.String(s.SecurityID)}
}
func (s *SecurityGroupRequest) CreatePortRequest() {
	iprange := []*ec2.IpRange{&ec2.IpRange{CidrIp: aws.String("0.0.0.0/0")}}
	permission := &ec2.IpPermission{
		FromPort:   aws.Int64(s.RequestPort),
		IpProtocol: aws.String("tcp"),
		IpRanges:   iprange,
		ToPort:     aws.Int64(s.RequestPort),
	}
	s.AuthorizeSecurityGroupIngressInput.IpPermissions = []*ec2.IpPermission{permission}
	s.RevokeSecurityGroupIngressInput.IpPermissions = []*ec2.IpPermission{permission}
}

func (s *SecurityGroupRequest) SecurityInfoSet() {
	svc := s.Ec2client
	input := &ec2.DescribeInstanceAttributeInput{
		Attribute:  aws.String("groupSet"),
		InstanceId: aws.String(s.InstanceID),
	}
	group, _ := svc.DescribeInstanceAttribute(input)
	s.SecurityID = *group.Groups[0].GroupId
	s.CreateRequest()
}
func (s *SecurityGroupRequest) OpenedList() {
	s.OpenPortList = []int64{}
	svc := s.Ec2client
	secinfo, _ := svc.DescribeSecurityGroups(s.DescribeSecurityGroupsInput)
	for _, i := range secinfo.SecurityGroups[0].IpPermissions {
		s.OpenPortList = append(s.OpenPortList, *i.FromPort)
	}
}

func (s *SecurityGroupRequest) OpenPort() error {
	svc := s.Ec2client
	p, err := svc.AuthorizeSecurityGroupIngress(s.AuthorizeSecurityGroupIngressInput)
	if err != nil {
		return errors.Wrap(err, "open port failed")
	}

	d.Info("port open:", s.RequestPort, p)
	return nil
}

func (s *SecurityGroupRequest) ClosePort() error {
	svc := s.Ec2client
	p, err := svc.RevokeSecurityGroupIngress(s.RevokeSecurityGroupIngressInput)
	if err != nil {
		return errors.Wrap(err, "close port failed")
	}

	d.Info("port close:", s.RequestPort, p)
	return nil
}

func Make(profilename string, awsRegion string, instanceID string) SecurityGroupRequest {
	req := SecurityGroupRequest{
		Sess:       session.Must(session.NewSession()),
		Cred:       credentials.NewSharedCredentials("/root/.aws/credentials", profilename),
		InstanceID: instanceID,
	}

	req.Ec2client = ec2.New(req.Sess, &aws.Config{
		Credentials: req.Cred,
		Region:      aws.String(awsRegion),
	})

	req.SecurityInfoSet()
	req.OpenedList()
	port, _ := private.GetFreePort()
	// req.RequestPort = req.random(1024, 65535)
	req.RequestPort = int64(port)
	req.CreatePortRequest()
	return req
}

func (s *SecurityGroupRequest) random(min int64, max int64) int64 {
	current := s.OpenPortList
	rand.Seed(time.Now().UnixNano())
	ret := rand.Int63n(max)
	if ret <= min {
		ret = s.random(min, max)
	} else if contains(current, ret) {
		ret = s.random(min, max)
	}
	return ret
}

func contains(list []int64, obj int64) bool {
	for _, o := range list {
		if obj == o {
			return true
		}
	}
	return false
}

//s3

func Download(env string, awsRegion string, profilename string) error {
	filename := []string{"fullchain.pem", "privkey.pem"}
	var myBucket string

	switch env {
	case "prod":
	case "dev":
		myBucket = "sesha3-dev"
	case "test":
		myBucket = "testsetty"
	default:
		return errors.New("invalid env value")
	}

	if myBucket == "" {
		return errors.New("bucket name cannot be empty")
	}

	sess := session.Must(session.NewSession())
	cred := credentials.NewSharedCredentials("/root/.aws/credentials", profilename)
	svc := s3.New(sess, &aws.Config{
		Credentials: cred,
		Region:      aws.String(awsRegion),
	})

	downloader := s3manager.NewDownloaderWithClient(svc)
	for _, i := range filename {
		fl := cmdline.Dir() + "/certs/" + i
		f, err := os.Create(fl)
		if err != nil {
			return errors.Wrapf(err, "create %s failed", fl)
		}

		// Write the contents of S3 Object to the file
		n, err := downloader.Download(f, &s3.GetObjectInput{
			Bucket: aws.String(myBucket),
			Key:    aws.String(i),
		})

		if err != nil {
			return errors.Wrap(err, "s3 download failed")
		}

		d.Info("download file:", i, "|", "bytes:", n)
	}

	return nil
}

//example code
//func main() {
//
//	const (
//		profilename  = "mobingi-yuto"
//		awsRegion    = "ap-northeast-1"
//		testinstance = "i-09094885155fee296"
//	)
//	req := Make(profilename, awsRegion, testinstance)
//	req.OpenedList()
//	fmt.Println(req.OpenPortList)
//	req.Openport()
//	req.OpenedList()
//	fmt.Println(req.OpenPortList)
//	req.Closeport()
//	req.OpenedList()
//	fmt.Println(req.OpenPortList)
//}
