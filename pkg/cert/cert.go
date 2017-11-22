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
	"github.com/mobingilabs/sesha3/pkg/constants"
	"github.com/mobingilabs/sesha3/pkg/notify"
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
	domain := util.Domain()
	zoneid := util.ZoneId()
	dns := util.GetPublicDns()

	// check once first, in case it's already done
	if validCname(domain, dns) {
		d.Info("already cnamed:", domain, dns)
		return domain, nil
	}

	var sess *session.Session
	var svc *route53.Route53

	if params.IsDev {
		sess = session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigDisable,
		}))

		svc = route53.New(sess, &aws.Config{
			Region: aws.String(util.GetRegion()),
		})
	} else {
		sess = session.Must(session.NewSession())
		cred := credentials.NewSharedCredentials(
			"/root/.aws/credentials",
			constants.SESHA3_ROUTE53_IAMPROFILE)

		svc = route53.New(sess, &aws.Config{
			Credentials: cred,
			Region:      aws.String(util.GetRegion()),
		})
	}

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

	m := "route53 (add/update): " + domain + " [cname] " + dns
	notify.HookPost(m)
	d.Info(m)

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

// SetupLetsEncryptCert attempts to install LetsEncrypt certificates locally using the instance id.
// This function assumes that certbot is already installed, and we are running under root account.
func SetupLetsEncryptCert(wait bool) error {
	_, err := AddLocalUrlToRoute53(wait)
	if err != nil {
		return errors.Wrap(err, "route53 add failed")
	}

	cmd := exec.Command(
		"/usr/local/bin/certbot",
		"certonly",
		"--standalone",
		"-d", util.Domain(),
		"--debug",
		"--quiet",
		"--agree-tos",
		"--email", "chew.esmero@mobingi.com")

	d.Info("cmd:", cmd.Args)
	_, err = cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "certbot failed")
	}

	return nil
}
