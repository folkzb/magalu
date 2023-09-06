terraform {
    required_providers {
        magalu = {
            version = "0.1"
            source = "magalucloud/mgc"
        }
    }
}

provider "magalu" {
    # This will be used later on to test the SDK loading functions
    apis = ["virtual-machine@1.60.0"]
}

resource "magalu_virtual-machine_instances" "myvm" {
  name = "my-tf-vm"
  type = "cloud-bs1.xsmall"
  desired_image = "cloud-ubuntu-22.04 LTS"
  key_name = "luizalabs-key"
  availability_zone = "br-ne-1c"
  status = "ACTIVE"
  allocate_fip = false
}

resource "magalu_block-storage_volume" "myvmvolume" {
    name = "myvmvolume"
    description = "myvmvolumedescription"
    size = 20
    desired_volume_type = "cloud_nvme"
}
