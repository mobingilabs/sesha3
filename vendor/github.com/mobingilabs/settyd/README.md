# settym
Secure tty of mobingi

put aws credential file on settym server `~/.aws/credentials`.

send json post (shell script)
```
pemkeyURL=$1
userIP="0.0.0.0"
stackID="mo-..."
json='{"pem":"'"${pemkeyURL}"'","user":"ec2-user","ip":"'"${userIP}"'","stackid":"'"${stackID}"'","timeout":"10"}'
curl -v -H "Content-Type: application/json" -d ${json} http://testyuto.labs.mobingi.com.:8080/json
```
