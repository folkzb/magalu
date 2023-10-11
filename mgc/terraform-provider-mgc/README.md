MGC Terraform
=============

## Provider

### Install

This will create an override to make terraform look for the provider in this
specific folder

```
./install.sh
```

### Build

Builds the provider to be used by the terraform application:

```sh
go build
```

### Example

To check if the provider is being correctly installed:

```
./run.sh provider plan
OR
./run.sh provider apply
```

It might be necessary to set the access_token TF variable.

> For now the access token is bigger than the amount of chars that is allowed in
the TF variable prompt, so use the TF_VAR_access_token to set the variable
value.

### Generate docs

To automatically generate the documentation for the Provider and it's Resources, run the command below. The documentation will be found at `terraform-provider-mgc/docs`.

```
cd ../../
./scripts/tf_generate_docs.sh
```

This script is integrated into the `build_release.sh` script. This ensures that all releases have up to date TF documentation

## OpenTofu
To use OpenTofu for install scripts, set the environment variable `MGC_OPENTF` to one

```
export MGC_OPENTF=1
```

## SDK

### OpenAPIs

To use the OAPI files with the terraform examples it's necessary to set where
the OAPI files are in relationship to the `*.tf` file:

```
export MGC_SDK_OPENAPI_DIR=../../../cli/openapis
```

Now we can use the `sdk.Group().GetChildByName()` function to retrieve the
elements of the file.
