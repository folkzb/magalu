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
  desired_status = "active"
}
