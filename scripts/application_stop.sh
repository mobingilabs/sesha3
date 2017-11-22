#!/bin/bash

if pgrep -x "sesha3" &> /dev/null; then
  /usr/local/bin/supervisorctl stop sesha3 &>> /home/ec2-user/codedeploy.log;
fi
