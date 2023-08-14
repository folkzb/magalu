terraform {
    required_providers {
        magalu = {
            version = "0.1"
            source = "magalucloud/mgc"
        }
    }
}

// TODO: For now I'm setting this up with the env var TF_VAR_access_token
// it's not working through the terminal in my laptop
variable "access_token" {
    sensitive = true
    type = string
    description = "Token used when authenticating in the MagaluCloud"
}

provider "magalu" {
    api_key = var.access_token
    # This will be used later on to test the SDK loading functions
    apis = ["virtual-machine@1.60.0"]
}

resource "magalu_virtual-machine_instances" "myvm" {
  name = "my-tf-vm"
  type = "cloud-bs1.xsmall"
  desired_image = "cloud-ubuntu-22.04 LTS"
  key_name = "luizalabs-key"
  availability_zone = "br-ne-1c"
  status = "active"
}

// This part is to test the resource Read function, replace id with existing
// values
import {
    to = magalu_virtual-machine_instances.read_res_vm
    id = "existing-vm-id"
}
resource "magalu_virtual-machine_instances" "read_res_vm" {
  name = "existing_instance"
  type = "cloud-bs1.xsmall"
  image = "cloud-ubuntu-22.04 LTS"
  key_name = "luizalabs-key"
  status = "shutoff"
}
