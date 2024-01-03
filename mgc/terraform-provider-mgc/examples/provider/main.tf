terraform {
    required_providers {
        mgc = {
            version = "0.1"
            source = "magalucloud/mgc"
        }
    }
}

provider "mgc" {}

resource "mgc_virtual-machine_instances" "myvm" {
  name = "my-tf-vm"
  machine_type = {
    name = "cloud-bs1.xsmall"
  }
  image = {
    name = "cloud-ubuntu-22.04 LTS"
  }
  key_name = "luizalabs-key"
  availability_zone = "br-ne-1c"
}

resource "mgc_block-storage_volume" "myvmvolume" {
    name = "myvmvolume"
    description = "myvmvolumedescription"
    size = 20
    volume_type = "cloud_nvme"
}

resource "mgc_block-storage_volume_attach" "myvmvolumeattachment" {
    id = mgc_block-storage_volume.myvmvolume.id
    virtual_machine_id = mgc_virtual-machine_instances.myvm.id
}
