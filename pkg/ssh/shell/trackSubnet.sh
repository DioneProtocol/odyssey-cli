#!/usr/bin/env bash
set -e
export PATH=$PATH:~/go/bin:~/.cargo/bin
/home/ubuntu/bin/odyssey subnet import file {{ .SubnetExportFileName }} --force
sudo systemctl stop odysseygo
/home/ubuntu/bin/odyssey subnet join {{ .SubnetName }} {{ .NetworkFlag }} --odysseygo-config /home/ubuntu/.odysseygo/configs/node.json --plugin-dir /home/ubuntu/.odysseygo/plugins --force-write
sudo systemctl start odysseygo
