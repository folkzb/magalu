#!/bin/bash

cat > ~/.terraformrc <<- EOM
provider_installation {
  dev_overrides {
    "registry.terraform.io/magalucloud/mgc" = "$PWD"
  }

  direct {}
}
EOM

echo "File .terraformrc writen to $HOME:"
cat ~/.terraformrc
