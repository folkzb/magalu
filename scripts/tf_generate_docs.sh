#!/bin/sh

set -xe

PROVIDER_DIR="./mgc/terraform-provider-mgc/"
TF_PLUGIN_DOCS="github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.16.0"
ABSOLUTE_PATH_OAPI=$(realpath ${MGC_SDK_OPENAPI_DIR:-./mgc/cli/openapis})

mkdir -p $PROVIDER_DIR/docs

MGC_SDK_OPENAPI_DIR=$ABSOLUTE_PATH_OAPI go run $TF_PLUGIN_DOCS generate --provider-dir=$PROVIDER_DIR
