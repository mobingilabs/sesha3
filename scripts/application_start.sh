#!/bin/bash

ln -sf /home/ubuntu/sesha3/supervisor.conf /etc/supervisor/conf.d/sesha3.conf
supervisorctl start sesha3 > /dev/null 2> /dev/null < /dev/null
