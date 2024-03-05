import re
from typing import (
    Any,
    Callable,
    Dict,
    List,
    Literal,
    NamedTuple,
    NotRequired,
    Optional,
    Tuple,
    TypeAlias,
    TypedDict,
    Union,
)
import jsonschema

ArgumentLocation: TypeAlias = Literal["query", "header", "path", "cookie"]
ArgumentStyle: TypeAlias = Literal[
    "matrix",
    "label",
    "form",
    "simple",
    "spaceDelimited",
    "pipeDelimited",
    "deepObject",
]

HttpMethod: TypeAlias = Literal["get", "put", "post", "delete", "patch"]

JSONSchema: TypeAlias = Dict[str, Any]  # TODO


class OAPIExample(TypedDict):
    summary: str | None
    description: str | None
    value: Any


class OAPIArgumentSchema(TypedDict):
    location: ArgumentLocation
    required: bool
    deprecated: bool
    description: str | None
    schema: JSONSchema
    examples: List[OAPIExample]
    style: ArgumentStyle
    explode: bool
    allow_empty_value: bool
    allow_reserved: bool


class OAPIHeaderSchema(TypedDict):
    required: bool
    deprecated: bool
    description: str | None
    schema: JSONSchema
    style: ArgumentStyle
    examples: List[OAPIExample]
    explode: bool


class MGCWaitTermination(TypedDict):
    interval: str
    maxRetries: int
    jsonPathQuery: NotRequired[str]
    templateQuery: NotRequired[str]


OAPILinkObject = TypedDict(
    "OAPILinkObject",
    {
        "operationId": NotRequired[str],
        "operationRef": NotRequired[str],
        "description": NotRequired[str],
        "parameters": Dict[str, Any],
        "requestBody": NotRequired[Any],
        "server": NotRequired[str],
        "x-mgc-hidden": NotRequired[bool],
        "x-mgc-wait-termination": NotRequired[MGCWaitTermination],
    },
)


class OAPIExampleObject(TypedDict):
    summary: NotRequired[str]
    description: NotRequired[str]
    value: NotRequired[Any]
    externalValue: NotRequired[str]


OAPIReferenceObject = TypedDict(
    "OAPIReferenceObject",
    {
        "$ref": str,
        "summary": NotRequired[str],
        "description": NotRequired[str],
    },
)


class OAPIContentSchema(TypedDict):
    schema: JSONSchema
    examples: List[OAPIExample]


class OAPIHeaderObject(TypedDict):
    description: NotRequired[str]
    required: NotRequired[bool]
    deprecated: NotRequired[bool]
    style: NotRequired[ArgumentStyle]
    explode: NotRequired[bool]
    schema: NotRequired[JSONSchema]
    example: NotRequired[Any]
    examples: NotRequired[Dict[str, OAPIExampleObject | OAPIReferenceObject]]
    content: NotRequired[Dict[str, "OAPIMediaTypeObject"]]


class OAPIEncodingObject(TypedDict):
    contentType: NotRequired[str]
    headers: NotRequired[Dict[str, OAPIHeaderObject | OAPIReferenceObject]]
    style: NotRequired[str]
    explode: NotRequired[bool]
    allowReserved: NotRequired[bool]


class OAPIMediaTypeObject(TypedDict):
    schema: NotRequired[JSONSchema | OAPIReferenceObject]
    example: NotRequired[Any]
    examples: NotRequired[Dict[str, OAPIExampleObject | OAPIReferenceObject]]
    encoding: NotRequired[Dict[str, OAPIEncodingObject]]


class OAPIResponseObject(TypedDict):
    description: str
    headers: Dict[str, OAPIHeaderSchema]
    content: Dict[str, OAPIMediaTypeObject]
    links: Dict[str, OAPILinkObject]


class OAPIRequestSchema(TypedDict):
    description: str | None
    required: bool
    content: Dict[str, OAPIContentSchema]


class OAPISecurityRequirement(TypedDict):
    name: str
    scopes: List[str]


# --- OAPI Specification (input)


class OAPIServerVariableObject(TypedDict):
    default: str
    description: NotRequired[str]
    enum: NotRequired[List[str]]


class OAPIServerObject(TypedDict):
    url: str
    description: NotRequired[str]
    variables: NotRequired[Dict[str, OAPIServerVariableObject]]


OAPITagObject = TypedDict(
    "OAPITagObject",
    {
        "name": str,
        "description": NotRequired[str],
        "x-mgc-name": NotRequired[str],
        "x-mgc-description": NotRequired[str],
        "x-mgc-hidden": NotRequired[bool],
    },
)


OAPIParameterObject = TypedDict(
    "OAPIParameterObject",
    {
        "name": str,
        "in": ArgumentLocation,  # NOTE: reserved keyword :-(
        "description": NotRequired[str],
        "required": NotRequired[bool],
        "deprecated": NotRequired[bool],
        "allowEmptyValue": NotRequired[bool],
        "style": NotRequired[str],
        "explode": NotRequired[bool],
        "allowReserved": NotRequired[bool],
        "schema": NotRequired[JSONSchema],
        "example": NotRequired[Any],
        "examples": NotRequired[Dict[str, OAPIExampleObject]],
        "content": NotRequired[Dict[str, OAPIMediaTypeObject]],
    },
)


class OAPIExternalDocumentationObject(TypedDict):
    url: str
    description: NotRequired[str]


class OAPIRequestBodyObject(TypedDict):
    description: NotRequired[str]
    content: Dict[str, OAPIMediaTypeObject]
    required: NotRequired[bool]


OAPIResponsesObject: TypeAlias = Dict[str, OAPIResponseObject | OAPIReferenceObject]

OAPISecurityRequirementObject: TypeAlias = Dict[str, List[str]]
OAPICallbackObject: TypeAlias = Dict[
    str, Union["OAPIPathItemObject", OAPIReferenceObject]
]


class OAPIOperationObject(TypedDict):
    tags: NotRequired[List[str]]
    summary: NotRequired[str]
    description: NotRequired[str]
    externalDocs: NotRequired[OAPIExternalDocumentationObject]
    operationId: NotRequired[str]
    parameters: NotRequired[List[OAPIParameterObject | OAPIReferenceObject]]
    requestBody: NotRequired[OAPIRequestBodyObject | OAPIReferenceObject]
    responses: NotRequired[OAPIResponsesObject]
    callbacks: NotRequired[Dict[str, OAPICallbackObject]]
    deprecated: NotRequired[bool]
    security: NotRequired[List[OAPISecurityRequirementObject]]
    servers: NotRequired[List[OAPIServerObject]]


OAPIPathItemObject = TypedDict(
    "OAPIPathItemObject",
    {
        "$ref": NotRequired[str],
        "summary": NotRequired[str],
        "description": NotRequired[str],
        "get": NotRequired[OAPIOperationObject],
        "put": NotRequired[OAPIOperationObject],
        "post": NotRequired[OAPIOperationObject],
        "delete": NotRequired[OAPIOperationObject],
        "options": NotRequired[OAPIOperationObject],
        "head": NotRequired[OAPIOperationObject],
        "patch": NotRequired[OAPIOperationObject],
        "trace": NotRequired[OAPIOperationObject],
        "servers": NotRequired[List[OAPIServerObject]],
        "parameters": NotRequired[List[OAPIParameterObject | OAPIReferenceObject]],
    },
)


class OAPIInfoObject(TypedDict):
    title: str
    version: str
    summary: NotRequired[str]
    description: NotRequired[str]


OAPISecuritySchemeApiKeyObject = TypedDict(
    "OAPISecuritySchemeApiKeyObject",
    {
        "type": Literal["apiKey"],
        "description": NotRequired[str],
        "name": str,
        "in": str,  # NOTE: reserved keyword :-(
    },
)


class OAPISecuritySchemeHttpObject(TypedDict):
    type: Literal["http"]  # noqa A003
    description: NotRequired[str]
    scheme: str
    bearerFormat: str


class OAPIOAuthFlowObject(TypedDict):
    authorizationUrl: str
    tokenUrl: str
    refreshUrl: NotRequired[str]
    scopes: Dict[str, str]


class OAPIOAuthFlowsObject(TypedDict):
    implicit: NotRequired[OAPIOAuthFlowObject]
    password: NotRequired[OAPIOAuthFlowObject]
    clientCredentials: NotRequired[OAPIOAuthFlowObject]
    authorizationCode: NotRequired[OAPIOAuthFlowObject]


class OAPISecuritySchemeOAuth2Object(TypedDict):
    type: Literal["oauth2"]  # noqa A003
    description: NotRequired[str]
    flows: OAPIOAuthFlowsObject


class OAPISecuritySchemeOpenIdConnectObject(TypedDict):
    type: Literal["openIdConnect"]  # noqa A003
    description: NotRequired[str]
    openIdConnectUrl: str


OAPISecuritySchemeObject: TypeAlias = (
    OAPISecuritySchemeApiKeyObject
    | OAPISecuritySchemeHttpObject
    | OAPISecuritySchemeOAuth2Object
    | OAPISecuritySchemeOpenIdConnectObject
)


class OAPIComponentsObject(TypedDict):
    schemas: NotRequired[Dict[str, JSONSchema]]
    responses: NotRequired[Dict[str, OAPIResponseObject | OAPIReferenceObject]]
    parameters: NotRequired[Dict[str, OAPIParameterObject | OAPIReferenceObject]]
    examples: NotRequired[Dict[str, OAPIExampleObject | OAPIReferenceObject]]
    requestBodies: NotRequired[Dict[str, OAPIRequestBodyObject | OAPIReferenceObject]]
    headers: NotRequired[Dict[str, OAPIHeaderObject | OAPIReferenceObject]]
    securitySchemes: NotRequired[
        Dict[str, OAPISecuritySchemeObject | OAPIReferenceObject]
    ]
    links: NotRequired[Dict[str, OAPILinkObject | OAPIReferenceObject]]
    callbacks: NotRequired[Dict[str, OAPICallbackObject | OAPIReferenceObject]]
    pathItems: NotRequired[Dict[str, OAPIPathItemObject | OAPIReferenceObject]]


OAPIObject = TypedDict(
    "OAPIObject",
    {
        "openapi": str,
        "info": OAPIInfoObject,
        "servers": NotRequired[List[OAPIServerObject]],
        "paths": NotRequired[Dict[str, OAPIPathItemObject]],
        "components": NotRequired[OAPIComponentsObject],
        "security": NotRequired[List[OAPISecurityRequirementObject]],
        "tags": NotRequired[List[OAPITagObject]],
        "externalDocs": NotRequired[OAPIExternalDocumentationObject],
        "$id": NotRequired[str],
    },
)


class OAPI(NamedTuple):
    path: str
    name: str
    obj: OAPIObject
    ref_resolver: jsonschema.RefResolver

    def resolve(self, ref: str) -> Any:
        return self.ref_resolver.resolve(ref)[1]


def is_ref(obj_or_ref: Any | OAPIReferenceObject) -> bool:
    return "$ref" in obj_or_ref


def get(obj_or_ref: Any | OAPIReferenceObject, resolve: Callable[[str], Any]) -> Any:
    if obj_or_ref is None:
        return None

    if is_ref(obj_or_ref):
        return resolve(obj_or_ref["$ref"])
    else:
        return obj_or_ref


# IMPORTANT: Everything below this is a direct recreation of the Golang code in
# 'operation_table.go'. It should be a 1:1 match


# If array is empty, returns None
def get_second_to_last_or_last_elem(arr: List[Any]) -> Optional[Any]:
    length = len(arr)
    if length == 1:
        return arr[0]
    if length > 1:
        return arr[length - 2]

    return None


# If slice is empty, returns None
def get_last_elem(arr: List[Any]) -> Optional[Any]:
    length = len(arr)
    if length == 0:
        return None

    return arr[length - 1]


class OperationDesc(TypedDict):
    path: str
    method: str
    op: OAPIOperationObject


class OperationTableEntry(TypedDict):
    name: List[str]
    variables: List[str]
    desc: OperationDesc
    key: str


def table_entry_simple_name_key(e: OperationTableEntry) -> str:
    return get_second_to_last_or_last_elem(e.get("name")) or ""


def table_entry_full_name_key(e: OperationTableEntry) -> str:
    length = len(e.get("name"))
    if length == 0:  # Should never happen
        return ""
    if length == 1:
        return e.get("name")[0]
    if length == 2:
        return e.get("name")[1] + "-" + e.get("name")[0]

    return (
        e.get("name")[length - 1]
        + "-"
        + e.get("name")[0]
        + "-"
        + e.get("name")[length - 2]
    )


class OperationTable(TypedDict):
    name: str
    child_tables: List["OperationTable"]
    child_operations: List[OperationTableEntry]


def find_table_entry_sibling(
    table: OperationTable, name: List[str]
) -> Tuple[int, OperationTableEntry | None]:
    for i, child_entry in enumerate(table.get("child_operations", [])):
        if child_entry["name"][0] == name[0] and len(child_entry["name"]) > 1:
            return i, child_entry

    return -1, None


def set_unique_full_keys(entries: List[OperationTableEntry]):
    max_var_length = -1
    for entry in entries:
        entry["key"] = table_entry_full_name_key(entry)

        var_length = len(entry.get("variables", []))
        if var_length > max_var_length:
            max_var_length = var_length

    for i in range(0, max_var_length):
        common_variable = ""
        is_common_variable = True

        for entry in entries:
            variables = entry.get("variables", [])
            if i > (len(variables) - 1):
                is_common_variable = False
                break

            if common_variable == "":
                common_variable = variables[i]
                continue

            if common_variable != variables[i]:
                is_common_variable = False
                break

        if is_common_variable:
            continue

        for entry in entries:
            variables = entry.get("variables", [])
            if i < len(variables):
                entry["key"] += "-" + variables[i]


def add_desc_to_table(
    table: OperationTable, name: List[str], variables: List[str], desc: OperationDesc
):
    if len(name) == 0:
        return

    for child_table in table.get("child_tables"):
        if child_table.get("name") == name[0]:
            add_desc_to_table(child_table, name[1:], variables, desc)
            return

    sibling_idx, sibling = find_table_entry_sibling(table, name)
    if sibling is not None:
        child_table = OperationTable(name=name[0], child_operations=[], child_tables=[])
        add_desc_to_table(
            child_table,
            sibling.get("name")[1:],
            sibling.get("variables"),
            sibling.get("desc"),
        )
        add_desc_to_table(child_table, name[1:], variables, desc)

        table.get("child_tables").append(child_table)
        table.get("child_operations").pop(sibling_idx)
        return

    entry = OperationTableEntry(name=name, variables=variables, desc=desc, key="")
    table.get("child_operations").append(entry)


def promote_op_table_to_parent(
    parent_table: OperationTable, child_table: OperationTable
):
    parent_table["child_tables"] = child_table.get("child_tables")
    parent_table["child_operations"] = child_table.get("child_operations")

    parent_table["name"] += "-" + child_table.get("name")


def simplify_op_table(table: OperationTable):
    for child_table in table.get("child_tables"):
        simplify_op_table(child_table)

    if len(table.get("child_operations")) == 0 and len(table.get("child_tables")) == 1:
        child_table = table.get("child_tables")[0]
        promote_op_table_to_parent(table, child_table)

    if len(table.get("child_operations")) == 1:
        entry = table.get("child_operations")[0]
        if name := get_last_elem(entry.get("name")):
            entry["name"] = [name]


def finalize_op_table_entry_keys(table: OperationTable):
    by_simple_key: Dict[str, List[OperationTableEntry]] = {}
    for child_operation in table.get("child_operations"):
        simple_key = table_entry_simple_name_key(child_operation)
        if name_ext := child_operation.get("desc").get("op").get("x-mgc-name"):
            simple_key = name_ext
        by_simple_key.setdefault(simple_key, []).append(child_operation)

    for simple_key, conflicting_entries in by_simple_key.items():
        if len(conflicting_entries) > 1:
            set_unique_full_keys(conflicting_entries)
        else:
            entry = conflicting_entries[0]
            if needs_full_name_key(entry.get("name")):
                entry["key"] = table_entry_full_name_key(entry)
            else:
                entry["key"] = simple_key

    for child_table in table.get("child_tables"):
        finalize_op_table_entry_keys(child_table)


names_that_need_to_be_prefixed_with_http_method = ["all", "default"]
http_methods_that_enforce_full_name = ["delete"]


def needs_full_name_key(name: List[str]) -> bool:
    return (
        get_second_to_last_or_last_elem(name)
        in names_that_need_to_be_prefixed_with_http_method
    )


open_api_path_arg_regex = re.compile("[{](?P<name>[^}]+)[}]")


def get_path_entry(path_entry: str) -> Tuple[str, bool]:
    match = open_api_path_arg_regex.match(path_entry)
    if match:
        if name := match.group("name"):
            return name or "", True
    return path_entry, False


def rename_http_method(http_method: str, ends_with_variable: bool) -> str:
    match http_method:
        case "post":
            return "create"
        case "put":
            return "replace"
        case "patch":
            return "update"
        case "get":
            if ends_with_variable:
                return "get"
            else:
                return "list"
    return http_method


def get_operation_name_and_variables(
    http_method: str, path_name: str
) -> Tuple[List[str], List[str]]:
    path_entries: List[str] = []
    variables: List[str] = []
    ends_with_variable = False

    for path_entry in path_name.split("/"):
        if not path_entry:
            continue

        variable, is_variable = get_path_entry(path_entry)
        if is_variable:
            variables.append(variable)
            ends_with_variable = True
        else:
            path_entries.extend(path_entry.split("-"))
            ends_with_variable = False

    path_entries.append(rename_http_method(http_method, ends_with_variable))

    return path_entries, variables


def new_operation_table(
    tag: OAPITagObject,
    oapi: OAPIObject,
) -> OperationTable:
    descs: List[OperationDesc] = []

    for path, path_item in oapi.get("paths", {}).items():
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

            if tag.get("name") not in op.get("tags", []):
                continue

            descs.append(OperationDesc(path=path, method=method, op=op))

    table = OperationTable(name=tag.get("name"), child_tables=[], child_operations=[])
    for desc in descs:
        desc_name, desc_variables = get_operation_name_and_variables(
            desc.get("method"), desc.get("path")
        )
        add_desc_to_table(table, desc_name, desc_variables, desc)

    simplify_op_table(table)
    finalize_op_table_entry_keys(table)
    table["name"] = tag.get("name")
    return table


def collect_operation_tables(
    oapi: OAPIObject,
) -> List[OperationTable]:
    tables: List[OperationTable] = []

    for tag in oapi.get("tags", {}):
        tables.append(new_operation_table(tag, oapi))

    return tables
