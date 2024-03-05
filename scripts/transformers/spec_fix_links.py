import argparse
from typing import Dict, Optional, cast
from links_helper import get_response_header, handle_exp
from spec_types import SpecTranformer
from oapi_types import OAPI, OAPIObject, OAPIOperationObject, OAPIResponseObject, get
from urllib import parse
import jsonpointer
from transform_helpers import fetch_and_parse, load_yaml, save_external

POSSIBLE_SOURCES = ["query", "header", "path", "body"]


class FixLinksTransformer(SpecTranformer):
    """Check if the id path in each link is valid"""

    def transform(self, oapi: OAPI):
        self._check_links_path(oapi)

    def _check_links_path(self, oapi: OAPI):
        """
        Adjusts response link IDs in the API spec

        Args:
            spec: The OpenAPI specification dictionary.
        """
        spec = oapi.obj
        for path, path_item in spec.get("paths", {}).items():
            path_ops: Dict[str, OAPIOperationObject | None] = {
                "get": path_item.get("get"),
                "post": path_item.get("post"),
                "put": path_item.get("put"),
                "patch": path_item.get("patch"),
                "delete": path_item.get("delete"),
            }

            for method, op in path_ops.items():
                if op is None:
                    continue
                for status_code, resp in op.get("responses", {}).items():
                    response: OAPIResponseObject = get(resp, oapi.resolve)

                    if status_code == "default" or status_code.startswith("2"):
                        if "links" in response:
                            for link in response.get("links", {}).values():
                                try:
                                    op_parameters = op.get("parameters", [])

                                    request_schema = self._get_content_app_json_schema(
                                        spec, cast(dict, op.get("requestBody", {}))
                                    )
                                    response_schema = self._get_content_app_json_schema(
                                        spec, cast(dict, response)
                                    )

                                    response_header = get_response_header(
                                        cast(dict, response)
                                    )

                                    for key, link_path in link.get(
                                        "parameters", {}
                                    ).items():
                                        field, result = handle_exp(
                                            link_path,
                                            request_schema,
                                            response_schema,
                                            response_header,
                                            op_parameters,
                                        )

                                        if result:
                                            continue
                                        if field is None:
                                            raise Exception(
                                                f"Found a invalid link in: "
                                                f"{path} - {method} - {link} - "
                                                f"{key}: {link['parameters'][key]}"
                                            )

                                        new_result = self.search_for_all_except_current(
                                            link_path,
                                            response_schema,
                                            request_schema,
                                            response_header,
                                            op_parameters,
                                            field,
                                        )
                                        if new_result:
                                            link["parameters"][key] = new_result
                                        else:
                                            raise Exception(
                                                f"Can't find a valid path for {key} "
                                                f"{path} - {method} - {op} "
                                            )
                                except KeyError:
                                    pass

    def _get_content_app_json_schema(
        self, spec: OAPIObject, sourceDict: dict
    ) -> Optional[Dict]:
        """
        Return the path of the schema refered in response content
        """
        schema = {}
        try:
            schema = sourceDict["content"]["application/json"]["schema"]
        except KeyError:
            return None

        if "$ref" in schema:
            schema_path = schema["$ref"]
            return jsonpointer.resolve_pointer(spec, schema_path[1:])
        else:
            return schema

    def search_for_all_except_current(
        self,
        current: str,
        responseBody: Optional[Dict],
        requestBody: Optional[Dict],
        responseHeader: dict,
        parameters: list,
        field: str,
    ):
        possible_parents = []
        if current.startswith("$request"):
            possible_parents = ["request", "response"]
        else:
            possible_parents = ["response", "request"]
        for parent in possible_parents:
            for s in POSSIBLE_SOURCES:
                new_exp = ""
                if s == "body":
                    new_exp = "$" + parent + "." + s + "#/" + field
                else:
                    new_exp = "$" + parent + "." + s + field
                if new_exp == current:
                    continue

                _, result = handle_exp(
                    new_exp,
                    requestBody,
                    responseBody,
                    responseHeader,
                    parameters,
                )
                if result is None:
                    continue
                else:
                    return result


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        prog="Check for OpenAPI links",
        description="Run through OpenAPI files check the path" "of each link parameter",
    )
    parser.add_argument(
        "spec_file", type=str, help="OpenAPI schema that need to be checked"
    )
    args = parser.parse_args()

    if parse.urlparse(args.spec_file).scheme != "":
        product_spec = fetch_and_parse(args.spec_file)
    else:
        product_spec = load_yaml(args.spec_file)

    instance = FixLinksTransformer()
    updated_spec = instance.transform(product_spec)
    save_external(product_spec, args.spec_file)
