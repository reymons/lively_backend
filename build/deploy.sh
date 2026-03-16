#!/bin/sh

set -eu
set -o pipefail

aws_access_key_id=$1
aws_secret_access_key=$2
aws_default_region=$3
aws_registry=$4
img_name=$5

echo "Authorizing AWS..."
sudo aws configure set aws_access_key_id $aws_access_key_id
sudo aws configure set aws_secret_access_key $aws_secret_access_key
sudo aws configure set default.region $aws_default_region
sudo aws ecr get-login-password \
    --region $aws_default_region | sudo docker login \
    --username AWS \
    --password-stdin $aws_registry > /dev/null 2>&1

echo "Pulling $img_name..."
sudo docker pull $img_name > /dev/null

echo "Starting the services..."
sudo docker-compose --env-file .env up --build -d

echo "Cleaning up the garbage..."
sudo docker system prune -f > /dev/null
sudo docker image prune --all -f > /dev/null
