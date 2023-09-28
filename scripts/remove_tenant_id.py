from typing import Any, Dict, List, Tuple
import argparse
import yaml


OAPISchema = Dict[str, Any]


def load_yaml(path: str) -> OAPISchema:
    with open(path, "r") as fd:
        return yaml.load(fd, Loader=yaml.CLoader)


def save_external(spec: OAPISchema, path: str):
    with open(path, "w") as fd:
        yaml.dump(spec, fd, sort_keys=False, indent=4, allow_unicode=True)


def filter_params_and_refs(
    params: List[str], spec: OAPISchema, param_name: str
) -> Tuple[List[str], List[str]]:
    refs = []
    filtered_params = []
    for p in params:
        pv = p
        ref = None

        if "$ref" in pv:
            ref = pv.get("$ref")
            ref_path = ref.removeprefix("#/").split("/")

            pv = spec
            for rp in ref_path:
                pv = pv[rp]

        if pv.get("name") != param_name:
            filtered_params.append(p)
        elif ref is not None:
            refs.append(ref)
            ref = None

    return filtered_params, refs


def remove_param_refs(spec: OAPISchema, refs: List[str]):
    def should_delete(value: Any, keys: List[str]):
        if len(keys) == 0:
            return True
        else:
            if should_delete(value[keys[0]], keys[1:]):
                del value[keys[0]]

            return len(value) == 0

    for ref in refs:
        ref_path = ref.removeprefix("#/").split("/")
        if should_delete(spec, ref_path):
            del spec[ref_path[0]]


def remove_param(spec: OAPISchema, param_name: str):
    refs_for_removal = set()
    for path in spec.get("paths", {}).values():
        for action in path.values():
            if not isinstance(action, dict) or "parameters" not in action:
                continue

            filtered_params, removable_refs = filter_params_and_refs(
                action.get("parameters", [{}]), spec, param_name
            )
            refs_for_removal.update(removable_refs)

            if not filtered_params:
                del action["parameters"]
            else:
                action["parameters"] = filtered_params

    remove_param_refs(spec, refs_for_removal)


def remove_tenant_id(spec: OAPISchema):
    remove_param(spec, param_name="x-tenant-id")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Remove `x-tenant-id` param from OpenAPI spec actions"
    )
    # External = Viveiro in MGC context, intermediate between product and Kong
    parser.add_argument(
        "path",
        type=str,
        help="File path to an OpenAPI spec to be parsed",
    )
    parser.add_argument(
        "-o",
        "--output",
        type=str,
        help="Path to save the modified YAML. Defaults to overwrite",
    )
    args = parser.parse_args()

    spec = load_yaml(args.path)

    remove_tenant_id(spec)

    save_external(spec, args.output or args.path)
