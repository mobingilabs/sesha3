package params

var (
	Environment string // dev or prod
	Domain      string // this server's domain
	Port        string // this server's port
	UseSyslog   bool   // use syslog for log library
	Region      string // aws region
	Ec2Id       string // this server's instance id
	CredProfile string // aws credenfile file profile
)
