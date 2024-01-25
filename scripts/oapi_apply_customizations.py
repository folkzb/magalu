import argparse
import os
import subprocess
import yaml
from typing import Any, Dict, TypedDict, Sequence

OAPISchema = Dict[str, Any]


class OAPIIndexModule(TypedDict):
    description: str
    name: str
    path: str
    summary: str
    url: str
    version: str


class OAPIIndex(TypedDict):
    modules: Sequence[OAPIIndexModule]
    version: str


def load_oapi_index(path: str) -> OAPIIndex:
    with open(path, "r") as fd:
        return yaml.load(fd, Loader=yaml.FullLoader)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        prog="Apply OAPI customizations",
        description="Run through OpenAPI files and apply the customization"
        "for each one",
    )
    # Internal = APIs generated directly from the code, always udpated
    parser.add_argument(
        "oapi_dir", type=str, help="Directory where OAPI files are stored"
    )
    parser.add_argument(
        "oapi_custom_dir",
        type=str,
        help="Directory where OAPI customization files are stored",
    )
    args = parser.parse_args()

    oapi_index = load_oapi_index(os.path.join(args.oapi_dir, "index.openapi.yaml"))
    script_dir = os.path.dirname(os.path.realpath(__file__))
    yaml_merge_script = os.path.join(script_dir, "yaml_merge.py")

    for module in oapi_index["modules"]:
        oapi_module = os.path.join(args.oapi_dir, module["path"])
        customizations = os.path.join(args.oapi_custom_dir, module["path"])

        subprocess.call(
            ["python3", yaml_merge_script, "--override", oapi_module, customizations]
        )
