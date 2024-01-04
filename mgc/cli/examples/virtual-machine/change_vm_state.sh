#!/bin/bash
set -xe

# Customize trace output for commands to stand out
PS4='\[\e[36m\]RAN COMMAND: \[\e[m\]'

MGC_CLI=${MGC_CLI:-./mgc}

# 1. Login
$MGC_CLI auth login

# 2. Creates VM and waits operation to be completed on server

IMAGE_ID="bc3dc4bc-4f9d-420d-99a7-702889dbcbaa"
KEY_NAME="luizalabs-key"
MACHINE_TYPE_ID="6343ae02-7c00-40ea-bfc8-5b998aeecc0c"
NAME="vm"
VPC_ID="ced3ed0a-044d-466c-9e9a-93fc8a78dba0"

read VM_ID < <($MGC_CLI virtual-machine-v1 instances create \
    --image "{ \"id\": \"${IMAGE_ID}\"}" \
    --ssh-key-name $KEY_NAME\
    --machine-type "{ \"id\": \"${MACHINE_TYPE_ID}\"}" \
    --name $NAME \
    --network "{ \"associate_public_ip\": true, \"interfaces\": [], \"vpc\": { \"id\": \"${VPC_ID}\" }}" \
    -o jsonpath='$.id')

read CUR_STATUS < <($MGC_CLI virtual-machine-v1 instances get \
    --id=$VM_ID \
	-U '30,10s,jsonpath=$.status=="running"')

# 3. Stops VM and waits for operation to be completed on server
$MGC_CLI virtual-machine-v1 instances stop --id $VM_ID

read CUR_STATUS < <($MGC_CLI virtual-machine-v1 instances get \
	--id=$VM_ID \
	-U '30,1s,jsonpath=$.state=="stopped"')

# 4. Starts VM and waits for operation to be completed on server
$MGC_CLI virtual-machine-v1 instances start --id $VM_ID

read CUR_STATUS < <($MGC_CLI virtual-machine-v1 instances get \
	--id=$VM_ID \
	-U '30,1s,jsonpath=$.state=="running"')

# 5. Suspends VM and waits operation to be completed on server
$MGC_CLI virtual-machine-v1 instances suspend --id $VM_ID

read CUR_STATUS < <($MGC_CLI virtual-machine-v1 instances get \
	--id=$VM_ID \
	-U '30,1s,jsonpath=$.state=="suspended"')

# 6. Clean up
$MGC_CLI virtual-machine-v1 instances delete --id $VM_ID
