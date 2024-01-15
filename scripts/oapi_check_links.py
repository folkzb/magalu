import argparse
import os
import subprocess
from oapi_apply_customizations import load_oapi_index


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        prog="Check OAPI links",
        description="Run through OpenAPI files and check the path of" "each link",
    )

    parser.add_argument(
        "oapi_dir", type=str, help="Directory where OAPI files are stored"
    )

    args = parser.parse_args()
    oapi_index = load_oapi_index(os.path.join(args.oapi_dir, "index.openapi.yaml"))
    script_dir = os.path.dirname(os.path.realpath(__file__))
    check_script = os.path.join(script_dir, "./transformers/spec_check_links.py")

    for module in oapi_index["modules"]:
        oapi_module = os.path.join(args.oapi_dir, module["path"])
        output_path = os.path.join(script_dir, "output.yaml")
        subprocess.call(["python3", check_script, oapi_module])
