package util

import (
	"github.com/spf13/cobra"
	"io/ioutil"
	"net/http"
)

func flag(cmd *cobra.Command, f string) string {
	s := cmd.Flag(f).DefValue
	if cmd.Flag(f).Changed {
		s = cmd.Flag(f).Value.String()
	}

	return s
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

func GetCliStringFlag(cmd *cobra.Command, f string) string {
	return flag(cmd, f)
}
