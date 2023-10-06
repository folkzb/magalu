#!/bin/sh

set -xe

PROVIDER_DIR="./mgc/terraform-provider-mgc/"
TF_PLUGIN_DOCS="github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.16.0"

mkdir -p $PROVIDER_DIR/docs

go run $TF_PLUGIN_DOCS generate --provider-dir=$PROVIDER_DIR
