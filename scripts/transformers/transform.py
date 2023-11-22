import argparse
import json
import yaml
from typing import List
from urllib import request, parse

from spec_types import OAPISchema, SpecTranformer
from spec_version_convert import ConvertVersionTransformer
from spec_remove_tenant_id import RemoveParamTransformer
from spec_update_error import UpdateErrorTransformer
from spec_remove_path import RemovePathTransformer
from spec_remove_component import RemoveComponentTransformer

from validate_openapi_specs import validate_oapi


def fetch_and_parse(json_oapi_url: str) -> OAPISchema:
    with request.urlopen(json_oapi_url, timeout=5) as response:
        return json.loads(response.read())


def load_yaml(path: str) -> OAPISchema:
    with open(path, "r") as fd:
        return yaml.load(fd, Loader=yaml.CLoader)


def save_external(spec: OAPISchema, path: str):
    with open(path, "w") as fd:
        yaml.dump(spec, fd, sort_keys=False, indent=4, allow_unicode=True)


def add_spec_uid(spec: OAPISchema, uid: str):
    spec["$id"] = uid


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        prog="Transform OAPI Spec",
        description="Transforms a product OAPI schema removing internal "
        "elements and making basic ajustments necessary to create a "
        '"public" schema that can be used by the MGC SDK.',
    )
    # Internal = APIs generated directly from the code, always udpated
    parser.add_argument(
        "spec_file",
        type=str,
        help="Raw product open API schema, can be yaml or json",
    )
    parser.add_argument(
        "spec_uid",
        type=str,
        help="Universal identifier for the specification, this will be used "
        "to identify operations between different products, should be an URL",
    )
    parser.add_argument(
        "-o",
        "--output",
        required=True,
        type=str,
        help="Path to save the new external YAML",
    )
    args = parser.parse_args()

    if parse.urlparse(args.spec_file).scheme != "":
        # If is a valid url fetch and load
        product_spec = fetch_and_parse(args.spec_file)
    else:
        # Load spec into dict
        product_spec = load_yaml(args.spec_file)

    # Perform changes in the spec
    transformers: List[SpecTranformer] = [
        ConvertVersionTransformer(),
        UpdateErrorTransformer(),
        RemovePathTransformer("/xaas"),
        RemoveComponentTransformer("xaas"),
        RemoveParamTransformer("x-tenant-id"),
    ]
    for t in transformers:
        product_spec = t.transform(product_spec)

    # Write external to output file
    add_spec_uid(product_spec, args.spec_uid)
    validate_oapi(product_spec)
    save_external(product_spec, args.output)
