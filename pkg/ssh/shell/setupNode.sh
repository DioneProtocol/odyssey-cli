#!/usr/bin/env bash
set -e
#name:TASK [update apt data and install dependencies]
DEBIAN_FRONTEND=noninteractive sudo apt-get -y update
DEBIAN_FRONTEND=noninteractive sudo apt-get -y install wget curl git
#name:TASK [create .odyssey-cli .odysseygo dirs]
mkdir -p .odyssey-cli .odysseygo/staking
#name:TASK [get odyssey go script]
wget -nd -m https://raw.githubusercontent.com/DioneProtocol/odyssey-docs/master/scripts/odysseygo-installer.sh
#name:TASK [modify permissions]
chmod 755 odysseygo-installer.sh
#name:TASK [call odyssey go install script]
./odysseygo-installer.sh --ip static --rpc private --state-sync on --fuji --version {{ .OdysseyGoVersion }}
#name:TASK [get odyssey cli install script]
wget -nd -m https://raw.githubusercontent.com/DioneProtocol/odyssey-cli/main/scripts/install.sh
#name:TASK [modify permissions]
chmod 755 install.sh
#name:TASK [run install script]
./install.sh -n
{{if .IsDevNet}}
#name:TASK [stop odysseygo in case of devnet]
sudo systemctl stop odysseygo
{{end}}
