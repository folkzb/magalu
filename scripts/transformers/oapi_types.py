from typing import (
    Any,
    Dict,
    List,
    Literal,
    NamedTuple,
    NotRequired,
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
