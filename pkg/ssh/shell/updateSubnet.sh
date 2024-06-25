#!/usr/bin/env bash
set -e
#name:TASK [stop node - stop odysseygo]
sudo systemctl stop odysseygo
#name:TASK [import subnet]
/home/ubuntu/bin/odyssey subnet import file {{ .SubnetExportFileName }} --force
#name:TASK [odyssey join subnet]
/home/ubuntu/bin/odyssey subnet join {{ .SubnetName }} --fuji --odysseygo-config /home/ubuntu/.odysseygo/configs/node.json --plugin-dir /home/ubuntu/.odysseygo/plugins --force-write
#name:TASK [restart node - start odysseygo]
sudo systemctl start odysseygo
