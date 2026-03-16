#!/bin/sh

set -eu

name=BackendStack
template=file://$PWD/deploy/template.yaml
key_name=main
key_filename=$PWD/key.pem
inst_name=backend

echo "Creating the stack $name using the template $template..."
aws cloudformation create-stack \
    --stack-name $name          \
    --template-body $template   \
    > /dev/null

aws cloudformation wait stack-create-complete --stack-name $name

echo "Getting EC2 public IP..."
ipv4_pub=$(aws ec2 describe-instances           \
    --filters "Name=tag:Name,Values=$inst_name" \
    --query "Reservations[0].Instances[0].PublicIpAddress" \
    --output text)

echo "Getting EC2 private key id..."
key_id=$(aws ec2 describe-key-pairs \
  --filters Name=key-name,Values=$key_name \
  --query "KeyPairs[0].KeyPairId" \
  --output text)

echo "Getting EC2 private key..."
aws ssm get-parameter           \
    --name /ec2/keypair/$key_id \
    --with-decryption           \
    --query Parameter.Value     \
    --output text               \
    > $key_filename
chmod 400 $key_filename

echo "Instance public IP is $ipv4_pub"
echo "Private key is located at $key_filename"
