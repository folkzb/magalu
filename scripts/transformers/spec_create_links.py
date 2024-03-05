from typing import (
    Callable,
    Mapping,
    Sequence,
    Tuple,
    TypedDict,
    Any,
    Dict,
    cast,
)
from copy import deepcopy
from links_helper import handle_exp
from spec_types import SpecTranformer
from oapi_types import (
    OAPI,
    JSONSchema,
    MGCWaitTermination,
    OAPIHeaderSchema,
    OAPILinkObject,
    OAPIOperationObject,
    OAPIParameterObject,
    OAPIReferenceObject,
    OAPIRequestBodyObject,
    OAPIResponseObject,
    OAPIResponsesObject,
    OperationTable,
    OperationTableEntry,
    collect_operation_tables,
    get,
)

POSSIBLE_PARENTS = ["request", "response"]
POSSIBLE_SOURCES = ["query", "header", "path", "body"]
METHODS_ALIAS = {
    "get": "get",
    "post": "create",
    "patch": "update",
    "put": "replace",
    "delete": "delete",
}

FROM_OPS_WITHOUT_LINKS = ["delete", "list"]
FROM_OPS_WITH_UPDATE_PROP_LINKS = ["get"]


class OperationFields(TypedDict):
    parameters: Sequence[OAPIParameterObject | OAPIReferenceObject]
    request_schema: JSONSchema | None
    response: OAPIResponseObject
    response_schema: JSONSchema | None
    response_headers: Mapping[str, OAPIHeaderSchema]
    operation_id: str
    description: str | None


class CreateLinks(SpecTranformer):
    """
    Organize operations into tables (same as final SDK tree), and generate links
    between siblings, filling in path parameter but leaving other parameters
    (header, query, cookie and requestBody) open for User definition when
    calling the link.

    When the link is to a 'get' operation and the source operation was updating
    a single property via requestBody that has a match in the 'get' operation's
    responseBody, a 'wait-termination' block is generated automatically.

    'update/prop' links are also generated, when the source operation is a 'get'
    one and the link is to an operation that updates a single prop (reverse case
    of the above paragraph)
    """

    def transform(self, oapi: OAPI):
        operation_tables = collect_operation_tables(oapi.obj)
        for table in operation_tables:
            self._generate_table_links(table, oapi.resolve)

    def _generate_table_links(
        self, table: OperationTable, resolve: Callable[[str], Any]
    ):
        for child_table in table.get("child_tables", {}):
            self._generate_table_links(child_table, resolve)

        for entry in table.get("child_operations", []):
            if not self._should_have_links(entry):
                continue

            for sibling in table.get("child_operations", []):
                if entry.get("key") == sibling.get("key"):
                    continue

                self._generate_op_link(table, entry, sibling, resolve)

    def _should_have_links(self, entry: OperationTableEntry) -> bool:
        for ignored in FROM_OPS_WITHOUT_LINKS:
            if entry.get("key", "").startswith(ignored):
                return False
        return True

    def _generate_op_link(
        self,
        table: OperationTable,
        entry: OperationTableEntry,
        sibling: OperationTableEntry,
        resolve: Callable[[str], Any],
    ):
        from_fields = self._collect_op_fields(
            entry.get("desc", {}).get("op", {}), resolve
        )
        if from_fields is None:
            return

        to_fields = self._collect_op_fields(
            sibling.get("desc", {}).get("op", {}), resolve
        )
        if to_fields is None:
            return

        link = self._create_standard_link(table, from_fields, to_fields, resolve)
        if link is None:
            return

        self._apply_link(link, from_fields["response"], sibling["key"])

        if entry.get("key") in FROM_OPS_WITH_UPDATE_PROP_LINKS:
            link, prop = self._create_update_prop_link(from_fields, to_fields, link)
            if link is not None:
                self._apply_link(link, from_fields["response"], "update/" + prop)

    def _create_standard_link(
        self,
        table: OperationTable,
        from_fields: OperationFields,
        to_fields: OperationFields,
        resolve: Callable[[str], Any],
    ) -> OAPILinkObject | None:
        parameters: Dict[str, str] = {}
        for parameter_or_ref in to_fields.get("parameters", []):
            parameter: OAPIParameterObject = get(parameter_or_ref, resolve)
            if parameter.get("in") != "path":
                continue

            param_name = parameter["name"]
            regexp: str | None = None

            for variant in self._param_name_variants(table, param_name):
                regexp = self._look_for_parameter(from_fields, variant)
                if regexp is not None:
                    break

            if regexp is None:
                continue

            parameters[param_name] = regexp

        if not parameters:
            return None

        link: OAPILinkObject = {
            "operationId": to_fields.get("operation_id", ""),
            "description": to_fields.get("description") or "",
            "parameters": parameters,
        }
        self._add_wait_termination(link, from_fields, to_fields)
        return link

    def _param_name_variants(
        self, table: OperationTable, param_name: str
    ) -> Sequence[str]:
        # 'the_resources' -> 'the_resource'
        res_name = self._table_name_singular(table)

        # param_name == the_resource_id
        return [
            # 'id' when res_name is 'the_resource'
            param_name.removeprefix(res_name + "_"),
            # 'id' when res_name is 'theresource'
            param_name.replace("_", "").removeprefix(res_name),
            # 'the_resource_id' raw
            param_name,
        ]

    def _create_update_prop_link(
        self,
        from_fields: OperationFields,
        to_fields: OperationFields,
        std_link: OAPILinkObject,
    ) -> Tuple[OAPILinkObject | None, str]:
        from_prop, to_prop = self._match_param_being_updated(from_fields, to_fields)
        if from_prop is None or to_prop is None:
            return None, ""

        link: OAPILinkObject = {
            "operationId": to_fields.get("operation_id", ""),
            "description": to_fields.get("description") or "",
            "parameters": deepcopy(std_link.get("parameters", {})),
        }
        link["x-mgc-hidden"] = True

        return link, from_prop

    def _add_wait_termination(
        self,
        link: OAPILinkObject,
        from_fields: OperationFields,
        to_fields: OperationFields,
    ):
        # from/to inverted here! We want to know if the link source is a prop
        # update and the link target is a subsequential "resource get". In this
        # case, 'to_fields' is the 'get' op and 'from_fields' is the 'update prop' op
        prop_in_get, prop_in_update = self._match_param_being_updated(
            to_fields, from_fields
        )
        if prop_in_get is None or prop_in_update is None:
            return

        link["x-mgc-wait-termination"] = MGCWaitTermination(
            interval="5s",
            maxRetries=10,
            jsonPathQuery="$.result."
            + prop_in_get
            + " == $.owner.parameters."
            + prop_in_update,
        )

    def _match_param_being_updated(
        self, from_fields: OperationFields, to_fields: OperationFields
    ) -> Tuple[str | None, str | None]:
        request_schema = to_fields.get("request_schema")
        if request_schema is None:
            return None, None

        request_props = request_schema.get("properties", {})
        if len(request_props) != 1:
            return None, None

        response_schema = from_fields.get("response_schema")
        if response_schema is None:
            return None, None

        response_props: Mapping[str, JSONSchema] = response_schema.get("properties", {})

        prop_name: str = next(iter(request_props))
        prop_name_sanitized = prop_name.removeprefix("new_")

        if prop_name_sanitized not in response_props:
            return None, None

        return prop_name_sanitized, prop_name

    def _apply_link(self, link: OAPILinkObject, response: OAPIResponseObject, key: str):
        current_link = response.setdefault("links", {}).setdefault(key, link)
        if current_link is link:
            return

        self._apply_dict(cast(dict, current_link), cast(dict, link))

    def _apply_dict(self, dst: dict, new: dict):
        for k, new_v in new.items():
            v = dst.setdefault(k, new_v)
            if v is new_v:
                continue

            if new_v is dict and v is dict:
                self._apply_dict(v, new_v)

    def _collect_op_fields(
        self, op: OAPIOperationObject, resolve: Callable[[str], Any]
    ) -> OperationFields | None:
        response = self._get_successful_response(op.get("responses", {}), resolve)
        if response is None:
            return None

        request_body: OAPIRequestBodyObject = get(op.get("requestBody"), resolve)

        if summary := op.get("summary"):
            description = summary
        else:
            description = op.get("description", "")

        return OperationFields(
            parameters=op.get("parameters", []),
            request_schema=self._get_json_content_schema(request_body, resolve),
            response=response,
            response_schema=self._get_json_content_schema(response, resolve),
            response_headers=response.get("headers", {}),
            operation_id=op.get("operationId", ""),
            description=description,
        )

    def _table_name_singular(self, table: OperationTable) -> str:
        return table.get("name", "").removesuffix("s")

    def _look_for_parameter(
        self,
        fields: OperationFields,
        param: str,
    ) -> str | None:
        for parent in POSSIBLE_PARENTS:
            for source in POSSIBLE_SOURCES:
                if source == "body":
                    regexp = "$" + parent + "." + source + "#/" + param
                else:
                    regexp = "$" + parent + "." + source + "." + param

                _, result = handle_exp(
                    regexp,
                    fields["request_schema"],
                    fields["response_schema"],
                    fields["response_headers"],
                    fields["parameters"],
                )
                if result is not None:
                    return result
        return None

    def _get_successful_response(
        self, responses: OAPIResponsesObject, resolve: Callable[[str], Any]
    ) -> OAPIResponseObject | None:
        for code_str, response in responses.items():
            code = int(code_str)
            if code < 200 or code > 299:
                continue

            return get(response, resolve)
        return None

    def _get_json_content_schema(
        self,
        source_dict: OAPIResponseObject | OAPIRequestBodyObject | None,
        resolve: Callable[[str], Any],
    ) -> JSONSchema | None:
        if source_dict is None:
            return None

        schema = (
            source_dict.get("content", {}).get("application/json", {}).get("schema")
        )
        if schema is None:
            return None

        return get(schema, resolve)
