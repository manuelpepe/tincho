#!/bin/bash -e
mkdir -p build
go build -o build/server cmd/server/main.go
ssh "$TINCHO_DEPLOY_USER@$TINCHO_DEPLOY_SERVER" "sudo systemctl stop tincho"
scp build/server "$TINCHO_DEPLOY_USER@$TINCHO_DEPLOY_SERVER:$TINCHO_DEPLOY_PATH/server"
ssh "$TINCHO_DEPLOY_USER@$TINCHO_DEPLOY_SERVER" "sudo systemctl start tincho"
