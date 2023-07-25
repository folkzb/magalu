# Overlays to Existing OpenAPI

These files should be applied on top of incoming OpenAPI in order
to add some extra keys such as `x-cli-name` to change the
command line interface name to use for a command or flag.

Apply using [yaml_merge.py](../scripts/yaml_merge.py), example:


```shell
cd ..
python3 scripts/yaml_merge.py \
    mgc/cli/openapis/vpc.openapi.yaml \
    openapi-customizations/vpc.openapi.yaml
```
