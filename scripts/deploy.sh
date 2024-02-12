#!/bin/bash -e
mkdir -p build
go build -o build/server cmd/server/main.go
scp build/server "$TINCHO_DEPLOY_USER@$TINCHO_DEPLOY_SERVER:$TINCHO_DEPLOY_PATH/server_new"
ssh -T "$TINCHO_DEPLOY_USER@$TINCHO_DEPLOY_SERVER" << EOF
sudo systemctl stop tincho
mv "$TINCHO_DEPLOY_PATH/server_new" "$TINCHO_DEPLOY_PATH/server"
sudo systemctl start tincho
EOF