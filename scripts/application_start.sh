#!/bin/bash

# append our [program] to supervisor if not present
grep 'sesha3' /etc/supervisord.conf || \
    echo -en '\n' >> /etc/supervisord.conf && \
    echo '[program:sesha3]' >> /etc/supervisord.conf && \
    echo 'command=/home/ec2-user/sesha3/sesha3 serve --syslog --rundev' >> /etc/supervisord.conf && \
    echo 'directory=/home/ec2-user/sesha3' >> /etc/supervisord.conf && \
    echo 'autostart=true' >> /etc/supervisord.conf && \
    echo 'autorestart=true' >> /etc/supervisord.conf && \
    echo 'stderr_logfile=syslog' >> /etc/supervisord.conf && \
    echo 'stdout_logfile=syslog' >> /etc/supervisord.conf

/usr/local/bin/supervisorctl reread &>> /home/ec2-user/codedeploy.log
/usr/local/bin/supervisorctl update &>> /home/ec2-user/codedeploy.log
/usr/local/bin/supervisorctl start sesha3 &>> /home/ec2-user/codedeploy.log
