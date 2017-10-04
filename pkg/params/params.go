package params

var (
	Domain      string // this server's domain
	UseSyslog   bool   // use syslog for log library
	Region      string // aws region
	Ec2Id       string // this server's instance id
	CredProfile string // aws credenfile file profile
)
