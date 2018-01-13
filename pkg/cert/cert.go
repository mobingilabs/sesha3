package cert

import (
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/golang/glog"
	"github.com/mobingilabs/sesha3/pkg/constants"
	"github.com/mobingilabs/sesha3/pkg/notify"
	"github.com/mobingilabs/sesha3/pkg/params"
	"github.com/mobingilabs/sesha3/pkg/util"
)

func validCname(domain, target string) bool {
	glog.V(1).Infof("check if %v points to %v...", domain, target)
	out, err := exec.Command("dig", domain).CombinedOutput()
	if err != nil {
		return false
	}

	glog.V(2).Infof("dig:out: %v", string(out))
	return strings.Contains(string(out), target)
}

func AddLocalUrlToRoute53(wait bool) (string, error) {
	domain := util.Domain()
	zoneid := util.ZoneId()
	dns := util.GetPublicDns()

	// check once first, in case it's already done
	if validCname(domain, dns) {
		glog.V(1).Infof("already cnamed: %v, %v", domain, dns)
		return domain, nil
	}

	var sess *session.Session
	var svc *route53.Route53

	if params.IsDev {
		// use ec2 role for credentials
		sess = session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigDisable,
		}))

		svc = route53.New(sess, &aws.Config{
			Region: aws.String(util.GetRegion()),
		})
	} else {
		// use the root credential file which is a different aws acct
		sess = session.Must(session.NewSession())
		svc = route53.New(sess, &aws.Config{
			Credentials: credentials.NewSharedCredentials(
				constants.ROOT_AWS_CRED_FILE,
				constants.SESHA3_ROUTE53_IAMPROFILE),
			Region: aws.String(util.GetRegion()),
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
		glog.Errorf("change recordset failed: %+v", util.ErrV(err))
		return domain, err
	}

	m := "route53 (add/update): " + domain + " [cname] " + dns
	notify.HookPost(m)
	glog.V(1).Info(m)

	if wait {
		// one day? why not?
		for i := 0; i < 720; i++ {
			if validCname(domain, dns) {
				glog.V(2).Infof("cname valid")
				break
			} else {
				glog.V(2).Infof("attempt # %v", i)
				time.Sleep(time.Second * 5)
			}
		}
	}

	glog.Infof("resp: %v", resp)
	return domain, nil
}

// SetupLetsEncryptCert attempts to install LetsEncrypt certificates locally using the instance id.
// This function assumes that certbot is already installed, and we are running under root account.
func SetupLetsEncryptCert(wait bool) error {
	_, err := AddLocalUrlToRoute53(wait)
	if err != nil {
		return util.ErrV(err, "route53 add failed")
	}

	cmd := exec.Command(
		"/usr/local/bin/certbot",
		"certonly",
		"--standalone",
		"-d", util.Domain(),
		"--debug",
		"--agree-tos",
		"--email", "chew.esmero@mobingi.com",
		"-n")

	glog.V(1).Infof("cmd: %v", cmd.Args)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return util.ErrV(err, "certbot failed")
	}

	glog.V(2).Infof("certbot out: %v", string(out))
	return nil
}
