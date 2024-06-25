#!/usr/bin/env bash
set -e
#name:TASK [upgrade odysseygo version]
./odysseygo-installer.sh --version {{ .OdysseyGoVersion }}
