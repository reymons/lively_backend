#!/bin/sh

name=BackendStack

echo "Deleting the stack $name..."
aws cloudformation delete-stack --stack-name $name > /dev/null
aws cloudformation wait stack-delete-complete --stack-name $name
echo "Done!"
