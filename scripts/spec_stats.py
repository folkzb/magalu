import os
from typing import Dict, Any, List, NamedTuple
import argparse
import yaml

OAPISchema = Dict[str, Any]
OAPIStats = Dict[str, List[Any]]


class OAPI(NamedTuple):
    path: str
    name: str
    schema: OAPISchema


class ResponseContext(NamedTuple):
    path: str
    method: str
    code: str


class OperationContext(NamedTuple):
    path: str
    method: str

    def key(self) -> str:
        return self.method.upper() + " " + self.path


class PathContext(NamedTuple):
    name: str


# This is used to fix list indentations, as Pyyaml doesn't indent them :/
class YamlDumper(yaml.Dumper):
    def increase_indent(self, flow=False, indentless=False):
        return super(YamlDumper, self).increase_indent(flow, False)


OPERATION_KEYS = ["get", "put", "post", "delete", "options", "head", "patch", "trace"]


def load_yaml(path: str) -> OAPISchema:
    with open(path, "r") as fd:
        return yaml.load(fd, Loader=yaml.CLoader)


def load_oapis(d: str) -> [OAPI]:
    schemas = []
    for f in os.listdir(d):
        name, ext = os.path.splitext(f)
        if name == "index" or ext != ".yaml":
            print("ignored file:", f)
            continue

        path = os.path.join(d, f)
        schemas.append(OAPI(name=name, path=path, schema=load_yaml(path)))

    return schemas


def fill_responses_stats(ctx: OperationContext, responses: OAPISchema, dst: OAPIStats):
    obj = {ctx.key(): []}

    for code, r in responses.items():
        content = r.get("content", {})
        if not content:
            # Return for now. In the future check for the code?
            return

        for t, _ in content.items():
            if t != "application/json":
                obj[ctx.key()].append({code: t})

    if obj[ctx.key()]:
        dst.setdefault("non-json-responses", []).append(obj)


def fill_req_body_stats(ctx: OperationContext, r: OAPISchema, dst: OAPIStats):
    content = r.get("content", {})
    if content:
        for t, _ in content.items():
            if t != "application/json":
                dst.setdefault("non-json-requests", []).append({ctx.key(): t})


def fill_operation_stats(ctx: OperationContext, o: OAPISchema, dst: OAPIStats):
    responses = o.get("responses", {})
    if responses:
        fill_responses_stats(ctx, responses, dst)

    req_body = o.get("requestBody", {})
    if req_body:
        fill_req_body_stats(ctx, req_body, dst)

    if "operationId" not in o:
        dst.setdefault("missing_operation_id", []).append(ctx.key())

    return


def fill_path_stats(ctx: PathContext, p: OAPISchema, dst: OAPIStats):
    for method, o in p.items():
        if isinstance(o, dict) and method in OPERATION_KEYS:
            op_ctx = OperationContext(path=ctx.name, method=method)
            fill_operation_stats(op_ctx, o, dst)
    return


def get_oapi_stats(o: OAPI) -> OAPIStats:
    result = {}

    for pn, p in o.schema.get("paths", {}).items():
        ctx = PathContext(name=pn)
        fill_path_stats(ctx, p, result)

    # TODO: Add stats for other fields

    return result


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Validate response and request bodies for all OAPI YAML"
        "files in directory"
    )
    parser.add_argument("dir", type=str, help="Directory of OpenAPI files")
    args = parser.parse_args()

    oapis = load_oapis(args.dir)
    for o in oapis:
        stats = get_oapi_stats(o)
        if stats:
            print(
                yaml.dump(
                    {o.name: stats}, Dumper=YamlDumper, indent=2, allow_unicode=True
                )
            )
