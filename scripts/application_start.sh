#!/bin/bash

# append our [program:sesha3] to supervisor if not present
grep -q 'sesha3' /etc/supervisord.conf || echo -e '\n[program:sesha3]\ncommand=/home/ec2-user/sesha3/sesha3 serve --syslog --rundev\ndirectory=/home/ec2-user/sesha3\nautostart=true\nautorestart=true\nstderr_logfile=syslog\nstdout_logfile=syslog' >> /etc/supervisord.conf

# replace command+args in supervisord.conf based on final args
sed -i -e '/^.*sesha3[[:blank:]]serve.*$/{r /home/ec2-user/sesha3/cmdline' -e 'd}' /etc/supervisord.conf &>> /home/ec2-user/codedeploy.log

# reload supervisor
/usr/local/bin/supervisorctl reread &>> /home/ec2-user/codedeploy.log
/usr/local/bin/supervisorctl update &>> /home/ec2-user/codedeploy.log
/usr/local/bin/supervisorctl start sesha3 &>> /home/ec2-user/codedeploy.log
