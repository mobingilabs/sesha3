#!/bin/bash

# append our [program] to supervisor if not present
grep -q -F '[program:sesha3]' foo.bar || \
    echo '' >> foo.bar && \
    echo '[program:sesha3]' >> foo.bar && \
    echo 'command=/home/ec2-user/sesha3/sesha3 serve --syslog' >> foo.bar && \
    echo 'directory=/home/ec2-user/sesha3' >> foo.bar && \
    echo 'autostart=true' >> foo.bar && \
    echo 'autorestart=true' >> foo.bar && \
    echo 'stderr_logfile=syslog' >> foo.bar && \
    echo 'stdout_logfile=syslog' >> foo.bar

/usr/local/bin/supervisorctl reread > /dev/null 2> /dev/null < /dev/null
/usr/local/bin/supervisorctl update > /dev/null 2> /dev/null < /dev/null
/usr/local/bin/supervisorctl start sesha3 > /dev/null 2> /dev/null < /dev/null
