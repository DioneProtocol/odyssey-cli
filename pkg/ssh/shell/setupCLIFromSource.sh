#!/usr/bin/env bash
set -e
export PATH=$PATH:~/go/bin
cd ~
rm -rf odyssey-cli
git clone --single-branch -b {{ .CliBranch }} https://github.com/DioneProtocols/odyssey-cli
cd odyssey-cli
./scripts/build.sh
cp bin/odyssey ~/bin/odyssey
