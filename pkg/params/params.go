package params

var (
	IsDev       bool   // dev or prod
	Port        string // this server's port
	UseSyslog   bool   // use syslog for log library
	Region      string // aws region
	Ec2Id       string // this server's instance id
	CredProfile string // aws credenfile file profile
)
