#!/bin/bash
set -xe

# Customize trace output for commands to stand out
PS4='\[\e[36m\]RAN COMMAND: \[\e[m\]'

MGC_CLI=${MGC_CLI:-./mgc}
SSH_KEY_NAME=$1

if [ "$SSH_KEY_NAME" == "" ]
then
    echo "SSH key name must be passed as argument"
    exit 1
fi

# 1. Login
$MGC_CLI auth login

# 3. Create VM
IMAGE_NAME="cloud-debian-11 LTS"
MACHINE_TYPE_NAME="cloud-bs1.xsmall"
INSTANCE_NAME="vm-example-1"
read VM_ID < <($MGC_CLI virtual-machine instances create \
    --image=name:"$IMAGE_NAME" \
    --machine-type=name:"$MACHINE_TYPE_NAME" \
    --key_name="$SSH_KEY_NAME" \
    --name="$INSTANCE_NAME" \
    -o="jsonpath=$.id")

# 4. Create Disk
DESCRIPTION="example-volume"
DISK_NAME="example-volume";
DISK_TYPE="cloud_nvme";
DISK_SIZE=1;
read DISK_ID < <($MGC_CLI block-storage volume create \
    --description=$DESCRIPTION \
    --name=$DISK_NAME \
    --volume-type=$DISK_TYPE \
    --size=$DISK_SIZE -o jsonpath='$.id')

# 5. Wait for the VM to transition to a terminal state (active, shutoff or error)
read ACTIVE_VM_ID < <($MGC_CLI virtual-machine instances get \
    $VM_ID \
    -U="30,1s,jsonpath=\$.status == \"completed\"" \
    -o jsonpath='$.id')

# 6. Check if VM is in active state
if [ "$VM_ID" != "$ACTIVE_VM_ID" ]
then
    $MGC_CLI virtual-machine instances delete --id=$VM_ID
    exit 1
else
    # 6. Attach Disk to VM - may fail if VM is in Pending status
    $MGC_CLI block-storage volume attach \
        --id=$DISK_ID \
        --virtual-machine-id=$VM_ID

    # 7. Shutoff VM
    $MGC_CLI virtual-machine instances update --id=$VM_ID --status="shutoff"
fi
