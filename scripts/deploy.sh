#!/bin/bash -e

DEPLOY_SERVER="$TINCHO_DEPLOY_SERVER"
DEPLOY_USER="$TINCHO_DEPLOY_USER"
DEPLOY_PATH="$TINCHO_DEPLOY_PATH"

if [[ -z "$DEPLOY_SERVER" || -z "$DEPLOY_USER" || -z "$DEPLOY_PATH" ]]; then
  echo "Missing environment variables"
  exit 1
fi

mkdir -p build
go build -o build/server cmd/server/main.go

scp build/server "$DEPLOY_USER@$DEPLOY_SERVER:$DEPLOY_PATH/server_new"
# shellcheck disable=SC2087
ssh -T "$DEPLOY_USER@$DEPLOY_SERVER" << EOF  
sudo systemctl stop tincho
mv "$DEPLOY_PATH/server_new" "$DEPLOY_PATH/server"
sudo systemctl start tincho
EOF