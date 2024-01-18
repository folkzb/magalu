import re
from typing import Optional
from spec_types import OAPISchema

FIND_VARIABLE_REGEX = r"/{(\w+)}$"


def extract_path(input_path) -> str:
    variable_match = re.search(FIND_VARIABLE_REGEX, input_path)
    if variable_match:
        return input_path[: variable_match.start()]

    remaining_path = input_path.rsplit("/", 1)[0]
    variable_before_match = re.search(FIND_VARIABLE_REGEX, remaining_path)
    if variable_before_match:
        return remaining_path + "/"

    return input_path


def handle_exp(
    path: str,
    request_schema: Optional[dict],
    response_schema: Optional[dict],
    response_header: dict,
    action_parameters: list,
):
    # TODO if necessary, we could handle the other expressions here
    # https://spec.openapis.org/oas/latest.html#runtime-expressions
    if path.startswith("$request."):

        def find_headers(field_name: str):
            return is_action_parameter_present(field_name, "header", action_parameters)

        return handle_source_exp(
            path,
            path.removeprefix("$request."),
            request_schema,
            find_headers,
            action_parameters,
        )
    if path.startswith("$response."):

        def find_headers(field_name: str):
            return is_header_present(field_name, response_header)

        return handle_source_exp(
            path,
            path.removeprefix("$response."),
            response_schema,
            find_headers,
            action_parameters,
        )
    return None, None


def handle_source_exp(
    entire_path, path, body, find_headers, action_parameters
) -> tuple[Optional[str], Optional[str]]:
    if path.startswith("path."):
        return get_rt_exp_path(
            entire_path,
            path.removeprefix("path"),
            action_parameters,
        )
    if path.startswith("body"):
        return get_rt_exp_body(
            entire_path,
            path.removeprefix("body"),
            body,
        )
    if path.startswith("header."):
        return get_rt_exp_header(
            entire_path,
            path.removeprefix("header"),
            find_headers,
        )
    if path.startswith("query."):
        return get_rt_exp_query(
            entire_path,
            path.removeprefix("query"),
            action_parameters,
        )
    return None, None


def get_rt_exp_path(
    entire_exp: str,
    field: str,
    action_parameters: list,
) -> tuple[str, Optional[str]]:
    field_in_parameters = is_action_parameter_present(
        field.removeprefix("."), "path", action_parameters
    )
    if field_in_parameters:
        return field, entire_exp
    return field, None


def get_rt_exp_body(
    entire_path,
    json_pointer: str,
    body: dict,
) -> tuple[str, Optional[str]]:
    field = get_field_from_json_pointer(json_pointer)

    if body and check_field_in_schema(field.removeprefix("."), body):
        new_path = build_path(entire_path, field.removeprefix("."))
        return field, new_path
    return field, None


def get_rt_exp_header(
    entire_exp: str,
    field: str,
    find_headers,
) -> tuple[str, Optional[str]]:
    field_in_parameters = find_headers(
        field.removeprefix("."),
    )
    if field_in_parameters:
        return field, entire_exp
    return field, None


def get_rt_exp_query(
    entire_exp: str,
    field: str,
    action_parameters: list,
) -> tuple[str, Optional[str]]:
    field_in_parameters = is_action_parameter_present(
        field.removeprefix("."), "query", action_parameters
    )

    if field_in_parameters:
        return field, entire_exp
    return field, None


def is_action_parameter_present(
    field_name: str, source: str, action_parameters: list
) -> bool:
    """
    Check for a field in action_parameters with specific source
    """
    if action_parameters:
        for obj in action_parameters:
            if obj["name"] == field_name and obj["in"] == source:
                return True

    return False


def get_field_from_json_pointer(json_pointer) -> str:
    json_pointer = json_pointer.removeprefix("#/")
    for i, char in enumerate(json_pointer):
        if char in ["[", "/"]:
            json_pointer = json_pointer[:i]
            break
    return json_pointer


def check_field_in_schema(field: str, schema: dict) -> bool:
    """
    Check for a field in response schema
    """
    if schema:
        try:
            if field in schema["properties"]:
                return True
        except KeyError:
            pass
    return False


def build_path(json_pointer: str, field: str) -> str:
    if "#/" in json_pointer:
        index = json_pointer.index("#/") + 2
        return json_pointer[:index] + field
    else:
        return json_pointer


def is_header_present(field_name: str, headers: dict) -> bool:
    if headers:
        if field_name in headers.values():
            return True
    return False


def get_response_header(spec: OAPISchema, response: dict) -> dict:
    return response.get("headers", {})
