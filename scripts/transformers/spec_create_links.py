from typing import List, Optional, Tuple
from spec_types import OAPISchema, SpecTranformer

METHODS_ALIAS = {
    "get": "get",
    "post": "create",
    "patch": "update",
    "put": "replace",
    "delete": "delete",
}
POSSIBLE_PARENTS = ["request", "response"]
POSSIBLE_SOURCES = ["query", "header", "path", "body"]


class CreateLinks(SpecTranformer):
    """Create links"""

    def transform(self, spec: OAPISchema) -> OAPISchema:
        self._generate_links(spec)
        return spec

    def _generate_links(self, spec: OAPISchema) -> None:
        all_paths = list(spec.get("paths", {}).keys())
        path_prefixes = self.get_path_prefixes(all_paths)

        pre_links = self.generate_pre_links(spec, all_paths, path_prefixes)

        for path, methods in spec.get("paths", {}).items():
            start, prefix = self.path_has_prefix(path, pre_links)
            if start:
                self.get_links_for_path(spec, methods, pre_links[prefix])

    def generate_pre_links(
        self, spec: OAPISchema, all_paths: List, prefixes: List
    ) -> dict:
        pre_links = {}
        for prefix in prefixes:
            links_for_prefix = self.get_pre_links(spec, all_paths, prefix)
            pre_links[prefix] = links_for_prefix

        return pre_links

    def get_links_for_path(self, spec: OAPISchema, methods: dict, pre_links: dict):
        for method, action in methods.items():
            if method == "delete":
                continue
            action_parameters = action.get("parameters", [])
            request_schema = self._get_content_app_json_schema(
                spec, action.get("requestBody", {})
            )
            # Check how get the response_schema withou going through responses
            action_response = action.get("responses", {})
            status = action_response.get("200", {})
            response_schema = self._get_content_app_json_schema(spec, status)
            response_header = self._get_response_header(spec, status)
            generated_links = {}
            for operation, value in pre_links.items():
                params = {}
                for param in value.get("parameters", {}):
                    result = self.search_for_path(
                        param["name"],
                        response_schema,
                        request_schema,
                        response_header,
                        action_parameters,
                    )
                    if result:
                        params[param["name"]] = result
                if params:
                    new_link = {
                        "operationId": value["operationId"],
                        "description": value["description"],
                        "parameters": params,
                    }
                    generated_links[operation] = new_link
            if generated_links:
                try:
                    for status in action["responses"].keys():
                        if status == "default" or status.startswith("2"):
                            action["responses"][status]["links"] = generated_links
                except KeyError:
                    pass

    def search_for_path(
        self,
        field_name,
        response_schema,
        request_schema,
        response_header,
        action_parameters,
    ):
        field_name = "." + field_name
        for parent in POSSIBLE_PARENTS:
            for s in POSSIBLE_SOURCES:
                new_exp = ""
                if s == "body":
                    new_exp = "$" + parent + "." + s + "#/" + field_name
                else:
                    new_exp = "$" + parent + "." + s + field_name
                field, result = self.handle_exp(
                    new_exp,
                    request_schema,
                    response_schema,
                    response_header,
                    action_parameters,
                )
                if result is None:
                    continue
                else:
                    return result
        pass

    def get_path_prefixes(self, paths: list):
        """
        Receives a list of all the paths from the openapi
        Return list of commom prefixes
        Example:
        receives ['<version>/<resource>/<variable>',
            '<version>/<resource>/<variable>/<action>]
        returns ['<version>/<resource>']

        With this list, we can map all the possible links to a specific resource
        and use it to define the links for each action
        """
        result_set = set()
        for path in paths:
            first_slash = path.find("/")
            if first_slash != -1:
                version = path.find("/", first_slash + 1)
                if version != -1:
                    resource = path.find("/", version + 1)
                    if resource != -1:
                        result_set.add(path[:resource])
                    else:
                        result_set.add(path[:resource])

        prefixes = list(result_set)
        return prefixes

    def get_pre_links(self, spec: OAPISchema, all_paths: List, prefix: str) -> dict:
        new_links = {}
        for path, value in spec.get("paths", {}).items():
            if path.startswith(prefix):
                for method, body in value.items():
                    try:
                        operationId = body["operationId"]
                        required_parameters = self.get_required_parameters(body)
                        # Confirm: if there is no parameters so there is no link?
                        """
                        here we must check for the requestBody
                        to confirm it exists
                        if yes, we must check for the get
                        to confirm if it return the the same data
                        if yes, create a update/<property_name>
                        """
                        if required_parameters:
                            key = self.generate_link_name(method, path)
                            new_obj = {
                                "operationId": operationId,
                                "description": body.get("description", {}),
                                "parameters": required_parameters,
                            }
                            new_links[key] = new_obj

                        requestBody = body.get("requestBody", None)
                        if requestBody:
                            (
                                is_valid,
                                property_name,
                            ) = self.validate_request_body_to_update(
                                spec, requestBody, prefix
                            )
                            if is_valid:
                                key = self.generate_update_link_name(property_name)
                                new_obj = {
                                    "operationId": operationId,
                                    "description": body.get("description", {}),
                                    "parameters": required_parameters,
                                }
                                new_links[key] = new_obj

                    except KeyError:
                        pass
        return new_links

    def generate_update_link_name(self, property_name: str) -> str:
        name = "update/" + property_name
        return name

    def get_required_parameters(self, body: dict):
        """
        Build the parameters for the link
        """
        required_parameters = [
            parameter
            for parameter in body.get("parameters", [])
            if parameter.get("required", True)
        ]

        return required_parameters

    def generate_link_name(self, method: str, path: str) -> str:
        """
        Note: This is very likeli to broke if
        there is a different pattern. Check it later
        Returns the name of the link
        Using this to avoid the error of overwriting
        every link with post method
        """
        if method.lower() == "post":
            id_index = path.rfind("{id}/")
            if id_index != -1:
                action = path[id_index + len("{id}/") :]
                return action.lower()
        link_name = METHODS_ALIAS.get(method, "default")
        return link_name

    def validate_request_body_to_update(
        self, spec: OAPISchema, requestBody: dict, prefix: str
    ):
        response_schema = {}
        for path, value in spec.get("paths", {}).items():
            if path.startswith(prefix) and path.endswith("}"):
                for method, body in value.items():
                    if method == "get":
                        try:
                            # TODO confirm is must check only in 200 status
                            response_schema = self._get_content_app_json_schema(
                                spec, body["responses"]["200"]
                            )
                        except KeyError:
                            pass

        props = list(response_schema.get("properties", {}).keys())

        if props:
            try:
                schema = self._get_content_app_json_schema(spec, requestBody)
                if self.is_valid_schema(schema):
                    name = list(schema["properties"].keys())
                    if name[0] in props:
                        return True, name[0]
            except KeyError:
                pass

        return False, None

    def is_valid_schema(self, schema):
        """
        We will create update links only for
        the methods that has a requestBody
        with only ONE parameter
        and that parameter is returned on the GET
        for the same path
        """
        return len(schema.get("properties", {}).keys()) == 1

    def _get_content_app_json_schema(self, spec: OAPISchema, sourceDict):
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

    def path_has_prefix(self, input_string: str, prefix_dict: dict) -> Tuple[bool, str]:
        for prefix in prefix_dict.keys():
            if input_string.startswith(prefix):
                return True, prefix
        return False, prefix

    def _get_response_header(self, spec: OAPISchema, response: dict):
        return response.get("headers", {})

    """
    TODO to validate the created link path i used some functions
    from spec_check_links.py. If the approach is okay
    i can it all to another file
    """

    def handle_exp(
        self,
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
    ):
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
    ):
        # TODO: find a way to solve the json_pointer for field
        # but that is not dependent of the data structure
        field = self.get_field_from_json_pointer(json_pointer)

        if body and self._check_field_in_schema(field.removeprefix("."), body):
            new_path = self.build_path(entire_path, field.removeprefix("."))
            return field, new_path
        else:
            return field, None

    def get_rt_exp_header(
        self,
        entire_exp: str,
        field: str,
        find_headers,
    ):
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
    ):
        field_in_parameters = self._is_action_parameter_present(
            field.removeprefix("."), "query", action_parameters
        )

        if field_in_parameters:
            return field, entire_exp
        else:
            return field, None

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

    def get_field_from_json_pointer(self, json_pointer):
        json_pointer = json_pointer.removeprefix("#/")
        for i, char in enumerate(json_pointer):
            if char in ["[", "/"]:
                json_pointer = json_pointer[:i]
                break

        return json_pointer

    def _check_field_in_schema(self, field: str, schema: dict) -> bool:
        """
        Check for a field in response schema
        """
        if schema:
            if field in schema.get("properties", {}):
                return True
        return False

    def build_path(self, json_pointer: str, field: str) -> str:
        if "#/" in json_pointer:
            index = json_pointer.index("#/") + 2
            return json_pointer[:index] + field
        else:
            return json_pointer

    def _is_header_present(self, field_name: str, headers: dict) -> bool:
        if headers:
            if field_name in headers.values():
                return True
        return False

    def _resolve_schema_path(self, spec: OAPISchema, path: str):
        # TODO define a proper solver for the schema path
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
