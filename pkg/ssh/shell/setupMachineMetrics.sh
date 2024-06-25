#!/usr/bin/env bash
#name:TASK [download monitoring script]
wget -nd -m https://raw.githubusercontent.com/DioneProtocol/odyssey-monitoring/main/grafana/monitoring-installer.sh
#name:TASK [modify permission for monitoring script]
chmod 755 monitoring-installer.sh
#name:TASK [set up Prometheus]
./monitoring-installer.sh --1
#name:TASK [set up node_exporter]
./monitoring-installer.sh --3
