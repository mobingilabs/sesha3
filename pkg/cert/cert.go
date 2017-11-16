package cert

import (
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/sesha3/pkg/params"
	"github.com/mobingilabs/sesha3/pkg/util"
	"github.com/pkg/errors"
)

func validCname(domain, target string) bool {
	d.Info("check if", domain, "points to", target)
	out, err := exec.Command("dig", domain).CombinedOutput()
	if err != nil {
		return false
	}

	d.Info("dig:out:", string(out))
	return strings.Contains(string(out), target)
}

func AddLocalUrlToRoute53(wait bool) (string, error) {
	iid := strings.Replace(util.GetEc2Id(), "-", "", -1)
	domain := "sesha3-" + iid + ".mobingi.com"
	if params.IsDev {
		domain = "sesha3-" + iid + ".labs.mobingi.com"
	}

	zoneid := "ZZDU2U8ZF5VZQ"
	if params.IsDev {
		zoneid = "Z23Y1M6Y77ZTL8"
	}

	dns := util.GetPublicDns()
	if dns == "" {
		return domain, errors.New("error in reading publicdns record")
	}

	sess := session.Must(session.NewSession())
	cred := credentials.NewSharedCredentials("/root/.aws/credentials", params.CredProfile)
	svc := route53.New(sess, &aws.Config{
		Credentials: cred,
		Region:      aws.String(params.Region),
	})

	r53p := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("UPSERT"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(domain),
						Type: aws.String("CNAME"),
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: aws.String(dns),
							},
						},
						TTL: aws.Int64(300),
					},
				},
			},
		},
		HostedZoneId: aws.String(zoneid),
	}

	resp, err := svc.ChangeResourceRecordSets(r53p)
	if err != nil {
		d.Error(err)
		return domain, err
	}

	if wait {
		// one day? why not?
		for i := 0; i < 720; i++ {
			if validCname(domain, dns) {
				break
			} else {
				time.Sleep(time.Second * 5)
			}
		}
	}

	d.Info(resp)
	return domain, nil
}

// SetupLetsEncryptCert attempts to install LetsEncrypt certificates locally using the instance id
// and registers it to Route53. This function assumes that certbot is already installed, and we
// are running under root account.
func SetupLetsEncryptCert() error {
	iid := strings.Replace(util.GetEc2Id(), "-", "", -1)
	domain := "sesha3-" + iid + ".labs.mobingi.com"
	if params.IsDev {
		domain = "sesha3-" + iid + ".mobingi.com"
	}

	d.Info("domain:", domain)
	cmd := exec.Command(
		"/usr/local/bin/certbot",
		"certonly",
		"--standalone",
		"-d",
		domain,
		"--debug",
		"--quiet",
		"--agree-tos",
		"--email",
		"--chew.esmero@mobingi.com")

	d.Info("cmd:", cmd.Args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "certbot failed")
	}

	d.Info("certbot:", string(out))
	return nil
}
