package util

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/golang/glog"
	"github.com/mobingilabs/sesha3/pkg/params"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// ErrV returns an error with stacktrace when glog level is 2, otherwise,
// return the same error with the optional wrap string, if provided.
func ErrV(err error, s ...string) error {
	if glog.V(2) {
		if len(s) > 0 {
			// add wrap text with stacktrace
			return errors.Wrap(err, s[0])
		}

		// add stacktrace
		return errors.WithStack(err)
	}

	if len(s) > 0 {
		return fmt.Errorf("%s: %v", s[0], err)
	}

	return err
}

func ZoneId() string {
	zoneid := "ZZDU2U8ZF5VZQ"
	if params.IsDev {
		zoneid = "Z1WHHSSMXMMSGW"
	}

	return zoneid
}

func Domain() string {
	iid := strings.Replace(GetEc2Id(), "-", "", -1)
	domain := "sesha3-" + iid + ".mobingi.com"
	if params.IsDev {
		domain = "sesha3-" + iid + ".demo.labs.mobingi.com"
	}

	return domain
}

func GetEc2Id() string {
	url := "http://169.254.169.254/latest/meta-data/instance-id"
	resp, _ := http.Get(url)
	defer resp.Body.Close()
	byteArray, _ := ioutil.ReadAll(resp.Body)
	return string(byteArray)
}

func GetRegion() string {
	url := "http://169.254.169.254/latest/meta-data/placement/availability-zone"
	resp, _ := http.Get(url)
	defer resp.Body.Close()
	byteArray, _ := ioutil.ReadAll(resp.Body)
	region := string(byteArray)[0 : len(string(byteArray))-1]
	return region
}

func GetPublicDns() string {
	url := "http://169.254.169.254/latest/meta-data/public-hostname"
	resp, _ := http.Get(url)
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)
	return string(b)
}

func GetCliStringFlag(cmd *cobra.Command, f string) string {
	return flag(cmd, f)
}

func flag(cmd *cobra.Command, f string) string {
	s := cmd.Flag(f).DefValue
	if cmd.Flag(f).Changed {
		s = cmd.Flag(f).Value.String()
	}

	return s
}
