#!/bin/bash

# append our [program:sesha3] to supervisor if not present
grep 'sesha3' /etc/supervisord.conf || echo -e '\n[program:sesha3]\ncommand=/home/ec2-user/sesha3/sesha3 serve --syslog --rundev\ndirectory=/home/ec2-user/sesha3\nautostart=true\nautorestart=true\nstderr_logfile=syslog\nstdout_logfile=syslog' >> /etc/supervisord.conf

# reload supervisor
/usr/local/bin/supervisorctl reread &>> /home/ec2-user/codedeploy.log
/usr/local/bin/supervisorctl update &>> /home/ec2-user/codedeploy.log
/usr/local/bin/supervisorctl start sesha3 &>> /home/ec2-user/codedeploy.log
