from typing import Any, Dict
import yaml
import argparse
import urllib.request
import json

OAPISchema = Dict[str, Any]


def sync_request_body(internal_spec: OAPISchema, external_spec: OAPISchema):
    for ext_path in external_spec.get("paths", {}):
        internal_path = internal_spec.get("paths", {}).get(ext_path)
        if not internal_path:
            # No problem, it was added to Kong but not in internal
            continue

        for ext_action in ext_path:
            internal_action = internal_path.get(ext_action)
            if not internal_action:
                # Action mapped on external but not on internal,
                # should never happen
                continue

            if internal_action["requestBody"]:
                ext_action["requestBody"] = internal_action["requestBody"]


def fetch_and_parse(json_oapi_url: str) -> OAPISchema:
    with urllib.request.urlopen(json_oapi_url, timeout=5) as response:
        return json.loads(response.read())


def load_yaml(path: str) -> OAPISchema:
    with open(path, "r") as fd:
        return yaml.load(fd, Loader=yaml.CLoader)


def save_external(spec: OAPISchema, path: str):
    with open(path, "w") as fd:
        yaml.dump(spec, fd, sort_keys=False, indent=4, allow_unicode=True)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        prog="SyncOAPI",
        description="Sync external OAPI schema with the internal schema by "
        "fixing any mismatch of requestBody between external and "
        "internal impl. After, we change the server URL to Kong and "
        "adjust schema of error returns. The YAML generated can "
        "be used in Kong directly to serve as a ref. to external.",
    )
    # Internal = APIs generated directly from the code, always udpated
    parser.add_argument(
        "internal_spec_url",
        type=str,
        help="URL to fetch current internal OpenAPI spec, which will "
        "come in JSON format",
    )
    parser.add_argument(
        "canonical_url",
        type=str,
        help="Canonical URL used to identify the spec",
    )
    # External = Viveiro in MGC context, intermediate between product and Kong
    parser.add_argument(
        "--ext",
        type=str,
        help="File path to current external OpenAPI spec. If not provided, downloaded "
        "internal spec will be used",
    )
    parser.add_argument(
        "-o",
        "--output",
        type=str,
        help="Path to save the new external YAML. Defaults to overwrite external spec",
    )
    args = parser.parse_args()

    # Load json into dict
    internal_spec = fetch_and_parse(args.internal_spec_url)
    # Load yaml into dict
    external_spec = load_yaml(args.ext) if args.ext else internal_spec

    # Replace requestBody from external to the internal value if they mismatch
    if args.ext:
        sync_request_body(internal_spec, external_spec)

    # Write external to file
    output_path = args.output or args.ext
    if output_path:
        save_external(external_spec, output_path)
