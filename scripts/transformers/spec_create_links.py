import re
from typing import Optional
from links_helper import extract_path, handle_exp
from spec_types import SpecTranformer
from oapi_types import OAPI
import jsonpointer

POSSIBLE_PARENTS = ["request", "response"]
POSSIBLE_SOURCES = ["query", "header", "path", "body"]
METHODS_ALIAS = {
    "get": "get",
    "post": "create",
    "patch": "update",
    "put": "replace",
    "delete": "delete",
}


class CreateLinks(SpecTranformer):
    """Create links"""

    def transform(self, oapi: OAPI):
        spec = oapi.obj
        all_paths = spec.get("paths", {})

        tree_root, level_paths, _ = build_tree(all_paths)
        related_paths = self.get_commom_parent_paths(level_paths)
        generated_links = self.generate_all_links(spec, related_paths, tree_root)

        for path, operations in spec.get("paths", {}).items():
            link = generated_links.get(path, {})
            if link:
                self.populate_action_links(spec, operations, link)

    def generate_all_links(self, spec, related_paths, tree_root) -> dict:
        generated_links = {}
        for level in related_paths.keys():
            for element in related_paths[level]:
                links_for_prefix = self.generate_links(spec, element, tree_root)
                generated_links[element["path"]] = links_for_prefix
        return generated_links

    def generate_links(self, spec, prefix, tree_root) -> dict:
        new_links = {}
        for path, operations in spec.get("paths", {}).items():
            if (
                path.startswith(prefix["prefix"])
                and find_path_level(tree_root, path) == prefix["level"]
            ):
                for method, operation in operations.items():
                    try:
                        operationId = operation["operationId"]
                        required_parameters = self.get_required_parameters(
                            spec, operation
                        )
                        requestBody = operation.get("requestBody", None)

                        if required_parameters:
                            key = self.generate_link_name(method, path)
                            new_obj = {
                                "operationId": operationId,
                                "description": operation.get("description", {}),
                                "parameters": required_parameters,
                            }
                            new_links[key] = new_obj

                        if "get" in operations and "post" in operations:
                            continue
                        if requestBody:
                            new_link_name, new_link = self.generate_update_link(
                                spec,
                                prefix,
                                path,
                                operationId,
                                operation,
                                required_parameters,
                            )

                            if new_link_name and new_link:
                                new_links[new_link_name] = new_link
                    except KeyError:
                        pass
        return new_links

    def generate_update_link(
        self,
        spec,
        prefix,
        path,
        operationId,
        operation,
        required_parameters,
    ) -> tuple[Optional[str], Optional[dict]]:
        (
            is_valid,
            property_name,
        ) = self.validate_request_body_to_update(
            spec, operation.get("requestBody", {}), prefix, path
        )
        if is_valid and property_name:
            key = self.generate_update_link_name(property_name)
            new_obj = {
                "operationId": operationId,
                "description": operation.get("description", {}),
                "parameters": required_parameters,
                "x-mgc-hidden": "true",
            }
            return key, new_obj
        return None, None

    def get_commom_parent_paths(self, level_nodes) -> dict:
        result_set: dict = {}
        for level, paths in level_nodes.items():
            result_set[level] = []
            for path in paths:
                result = extract_path(path)
                obj = {"level": level, "prefix": result, "path": path}
                result_set.setdefault(level, []).append(obj)

        return result_set

    def get_required_parameters(self, spec, operation: dict) -> list:
        """
        Build the parameters for the link
        """
        required_parameters = []
        try:
            params = operation.get("parameters", {})

            for param in params:
                if "$ref" in param:
                    schema = jsonpointer.resolve_pointer(spec, param["$ref"][1:])
                    if schema.get("in") == "path":
                        required_parameters.append(schema)

                else:
                    for parameter in operation.get("parameters", []):
                        if parameter.get("in") == "path":
                            required_parameters.append(parameter)

        except KeyError:
            pass

        return required_parameters

    def generate_link_name(self, method: str, path: str) -> str:
        if method.lower() == "post":
            id_index = path.rfind("id}/")
            if id_index != -1:
                action = path[id_index + len("id}/") :]
                return action.split("/")[0].lower()

        path_parts = path.strip("/").split("/")
        last_part = path_parts[-1]
        if "{" in last_part and "}" in last_part:
            path_parts.pop()
            last_part = path_parts[-1]

        link_name = METHODS_ALIAS.get(method.lower(), "default")

        return link_name

    def validate_request_body_to_update(
        self, spec: OAPISchema, requestBody: dict, prefix: str, given_path: str
    ) -> tuple[bool, Optional[str]]:
        if not requestBody:
            return False, None
        has, path_prefix = self.check_path_variable(given_path)
        if not has:
            return False, None

        response_schema: Optional[dict] = {}
        for path, value in spec.get("paths", {}).items():
            if path.startswith(path_prefix) and path.endswith("}"):
                for method, operation in value.items():
                    if method == "get":
                        try:
                            success_response = self.find_first_success_response(
                                operation.get("responses", {})
                            )

                            if not success_response:
                                return False, None

                            response_schema = self._get_content_app_json_schema(
                                spec, success_response
                            )

                        except KeyError:
                            pass
        if response_schema:
            props_list = list(response_schema.get("properties", {}).keys())
            properties = response_schema.get("properties", {})
        if props_list:
            try:
                schema = self._get_content_app_json_schema(spec, requestBody)
                if not schema:
                    return False, None
                if self.is_valid_schema(schema):
                    name = list(schema["properties"].keys())
                    if (
                        name[0] in props_list
                        and properties[name[0]]["type"] == "string"
                    ):
                        return True, name[0]
            except KeyError:
                pass
        return False, None

    def find_first_success_response(self, operations) -> Optional[dict]:
        for status, operation in operations.items():
            if status.startswith("2"):
                return operation
        return None

    def is_valid_schema(self, schema) -> bool:
        """
        We will create update links only for
        the methods that has a requestBody
        with only ONE parameter
        and that parameter is returned on the GET
        for the same path
        """
        return len(schema.get("properties", {}).keys()) == 1

    def _get_content_app_json_schema(
        self, spec: OAPISchema, sourceDict
    ) -> Optional[dict]:
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

    def check_path_variable(self, path: str) -> tuple[bool, Optional[str]]:
        match = re.search(r"(.*)/\{[^/]+\}$", path)
        if match:
            return False, match.group(1)
        else:
            remaining_path = path.rsplit("/", 1)[0]

            remaining_match = re.search(r"(.*)/\{[^/]+\}$", remaining_path)
            if remaining_match:
                return True, remaining_path

        return False, None

    def generate_update_link_name(self, property_name: str) -> str:
        name = "update/" + property_name
        return name

    def populate_action_links(
        self, spec: OAPISchema, operations: dict, pre_links: dict
    ):
        for method, operation in operations.items():
            if method == "delete":
                continue
            action_parameters = self.resolve_params(spec, operation)
            request_schema = self._get_content_app_json_schema(
                spec, operation.get("requestBody", {})
            )
            success_response = self.find_first_success_response(
                operation.get("responses", {})
            )

            if not success_response:
                return
            response_schema = self._get_content_app_json_schema(spec, success_response)
            response_header = self.get_response_header(success_response)
            generated_links = {}
            for link_name, value in pre_links.items():
                params = {}
                if value.get("operationId") == operation.get("operationId"):
                    continue
                for param in value["parameters"]:
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
                    if value.get("x-mgc-hidden", {}) == "true":
                        new_link.setdefault("x-mgc-hidden", True)
                    generated_links[link_name] = new_link
            if generated_links:
                try:
                    for status in operation["responses"].keys():
                        if status == "default" or status.startswith("2"):
                            operation["responses"][status]["links"] = generated_links
                except KeyError:
                    pass

    def resolve_params(self, spec, operation) -> list:
        """
        Build the parameters for the link
        """
        required_parameters = []
        try:
            params = operation.get("parameters", {})
            for param in params:
                if "$ref" in param:
                    schema_path = param["$ref"]
                    schema = jsonpointer.resolve_pointer(spec, schema_path[1:])
                    required_parameters.append(schema)
                else:
                    required_parameters.append(param)

        except KeyError:
            pass
        return required_parameters

    def get_response_header(self, response: dict) -> dict:
        return response.get("headers", {})

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
                _, result = handle_exp(
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


class TreeNode:
    def __init__(self, name):
        self.name = name
        self.children = {}
        self.paths = []


def build_tree(
    paths,
) -> tuple[TreeNode, dict[int, list[str]], dict[int, list[tuple[TreeNode, str]]]]:
    """
    It receives a list of paths and build a tree from its components
    Each path is parsed by / and each result part becomes a node
    """
    root = TreeNode("/")
    level_paths: dict = {}  # Dictionary to store paths at each level
    level_nodes: dict[int, list[tuple[TreeNode, str]]] = {0: [(root, "")]}

    for path in paths:
        current_node = root
        components = path.split("/")[1:]

        for level, component in enumerate(components, start=1):
            if component not in current_node.children:
                new_node = TreeNode(component)
                current_node.children[component] = new_node

                level_paths.setdefault(level, []).append(path)
                level_nodes.setdefault(level, []).append((new_node, path))

            else:
                current_node.paths.append(path)

            current_node = current_node.children[component]

    return root, level_paths, level_nodes


def find_path_level(root, path) -> int:
    current_node = root
    components = path.split("/")[1:]

    for level, component in enumerate(components, start=1):
        if component in current_node.children:
            current_node = current_node.children[component]
        else:
            break

    return level


def print_tree(node, indent=0) -> None:
    """
    Helper function to visualize the generated tree
    You can call it by passing the root node returned
    by build_tree
    """
    print("  " * indent + node.name)
    for child in node.children.values():
        print_tree(child, indent + 1)
