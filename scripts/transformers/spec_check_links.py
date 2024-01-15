import argparse
from typing import Dict, Optional, Tuple, Union
from spec_types import OAPISchema, SpecTranformer
from urllib import parse

from transform_helpers import fetch_and_parse, load_yaml, save_external

POSSIBLE_SOURCES = ["query", "header", "path", "body"]


class FixLinksTransformer(SpecTranformer):
    """Check if the id path in each link is valid"""

    def transform(self, spec: OAPISchema) -> OAPISchema:
        self._check_links_path(spec)
        return spec

    def _check_links_path(self, spec: OAPISchema) -> None:
        """
        Adjusts response link IDs in the API spec

        Args:
            spec: The OpenAPI specification dictionary.
        """
        for path, methods in spec.get("paths", {}).items():
            for method, action in methods.items():
                for status_code, response in action.get("responses", {}).items():
                    if status_code == "default" or status_code.startswith("2"):
                        if "links" in response:
                            for op, link in response["links"].items():
                                try:
                                    action_parameters = action.get("parameters")

                                    request_schema = self._get_content_app_json_schema(
                                        spec, action.get("requestBody", {})
                                    )
                                    response_schema = self._get_content_app_json_schema(
                                        spec, response
                                    )

                                    response_header = self._get_response_header(
                                        spec, response
                                    )

                                    for key, link_path in link["parameters"].items():
                                        field, result = self.handle_exp(
                                            link_path,
                                            request_schema,
                                            response_schema,
                                            response_header,
                                            action_parameters,
                                        )

                                        if result:
                                            continue
                                        if field is None:
                                            raise Exception(
                                                f"Found a invalid link in: "
                                                f"{path} - {method} - {op} - "
                                                f"{key}: {link['parameters'][key]}"
                                            )

                                        new_result = self.search_for_all_except_current(
                                            link_path,
                                            response_schema,
                                            request_schema,
                                            response_header,
                                            action_parameters,
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
        self, spec: OAPISchema, sourceDict: dict
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
            return self._resolve_schema_path(spec, schema_path)
        else:
            return schema

    def _get_response_header(self, spec: OAPISchema, response: dict):
        return response.get("headers", {})

    def handle_exp(
        self,
        path: str,
        request_schema: Optional[Dict],
        response_schema: Optional[Dict],
        response_header: dict,
        action_parameters: list,
    ):
        # TODO if necessary, we could handle the other expressions here
        # https://spec.openapis.org/oas/latest.html#runtime-expressions
        if path.startswith("$request."):

            def find_headers(field_name: str):
                return self._is_action_parameter_present(
                    field_name, "header", action_parameters
                )

            return self.handle_source_exp(
                path,
                path.removeprefix("$request."),
                request_schema,
                find_headers,
                action_parameters,
            )
        elif path.startswith("$response."):

            def find_headers(field_name: str):
                return self._is_header_present(field_name, response_header)

            return self.handle_source_exp(
                path,
                path.removeprefix("$response."),
                response_schema,
                find_headers,
                action_parameters,
            )
        else:
            return None, None

    def handle_source_exp(
        self, entire_path, path, body, find_headers, action_parameters
    ):
        if path.startswith("path."):
            return self.get_rt_exp_path(
                entire_path,
                path.removeprefix("path"),
                action_parameters,
            )
        if path.startswith("body"):
            return self.get_rt_exp_body(
                entire_path,
                path.removeprefix("body"),
                body,
            )
        if path.startswith("header."):
            return self.get_rt_exp_header(
                entire_path,
                path.removeprefix("header"),
                find_headers,
            )
        if path.startswith("query."):
            return self.get_rt_exp_query(
                entire_path,
                path.removeprefix("query"),
                action_parameters,
            )

    def get_rt_exp_path(
        self,
        entire_exp: str,
        field: str,
        action_parameters: list,
    ) -> Tuple[str, Union[str, None]]:
        field_in_parameters = self._is_action_parameter_present(
            field.removeprefix("."), "path", action_parameters
        )
        if field_in_parameters:
            return field, entire_exp
        else:
            return field, None

    def get_rt_exp_body(
        self,
        entire_path,
        json_pointer: str,
        body: dict,
    ) -> Tuple[str, Union[str, None]]:
        # TODO: find a way to solve the json_pointer for field
        # but that is not dependent of the data structure
        field = self.get_field_from_json_pointer(json_pointer)

        if body and self._check_field_in_schema(field.removeprefix("."), body):
            new_path = self.build_path(entire_path, field.removeprefix("."))
            return field, new_path
        else:
            return field, None

    def build_path(self, json_pointer: str, field: str) -> str:
        if "#/" in json_pointer:
            index = json_pointer.index("#/") + 2
            return json_pointer[:index] + field
        else:
            return json_pointer

    def get_rt_exp_header(
        self,
        entire_exp: str,
        field: str,
        find_headers,
    ) -> Tuple[str, Union[str, None]]:
        field_in_parameters = find_headers(
            field.removeprefix("."),
        )
        if field_in_parameters:
            return field, entire_exp
        else:
            return field, None

    def get_rt_exp_query(
        self,
        entire_exp: str,
        field: str,
        action_parameters: list,
    ) -> Tuple[str, Union[str, None]]:
        field_in_parameters = self._is_action_parameter_present(
            field.removeprefix("."), "query", action_parameters
        )

        if field_in_parameters:
            return field, entire_exp
        else:
            return field, None

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

                field, result = self.handle_exp(
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

    def get_field_from_json_pointer(self, json_pointer):
        json_pointer = json_pointer.removeprefix("#/")
        for i, char in enumerate(json_pointer):
            if char in ["[", "/"]:
                json_pointer = json_pointer[:i]
                break

        return json_pointer

    def _is_header_present(self, field_name: str, headers: dict) -> bool:
        if headers:
            if field_name in headers.values():
                return True
        return False

    def _is_action_parameter_present(
        self, field_name: str, source: str, action_parameters: list
    ) -> bool:
        """
        Check for a field in action_parameters with specific source
        """
        if action_parameters:
            for obj in action_parameters:
                if obj["name"] == field_name and obj["in"] == source:
                    return True
        return False

    def _check_field_in_schema(self, field: str, schema: Dict) -> bool:
        """
        Check for a field in response schema
        """
        if schema:
            if field in schema["properties"]:
                return True
        return False

    def _resolve_schema_path(self, spec: Dict, path: str) -> Optional[Dict]:
        """
        Resolves the path to a schema.

        Args:
            spec: the openapi definition
            path: the path of the schema defined in $ref.
            Example: '#/components/schemas/<name>'

        Returns:
            The schema definition dictionary, or None if not found.
        """
        if not path:
            return None

        components = path.split("/")

        if components[0] == "#":
            components = components[1:]

        current_dict = spec
        for component in components:
            if component in current_dict:
                current_dict = current_dict[component]
            else:
                return None

        return current_dict


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
