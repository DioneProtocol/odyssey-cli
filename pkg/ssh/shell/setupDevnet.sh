#!/usr/bin/env bash
set -e
#name:TASK [stop node]
sudo systemctl stop odysseygo
#name:TASK [remove previous odysseygo db and logs]
rm -rf /home/ubuntu/.odysseygo/db/
rm -rf /home/ubuntu/.odysseygo/logs/
#name:TASK [start node]
sudo systemctl start odysseygo
