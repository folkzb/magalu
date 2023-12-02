from typing import Any

from spec_types import OAPISchema, SpecTranformer


class AddParameterTypes(SpecTranformer):
    """If parameter schema type is unset, set it to 'string'

    https://spec.openapis.org/oas/latest.html#parameterObject
    defines a schema, but it's not required. Since the usage implies
    a string, let's set it to that.
    """

    def transform(self, spec: OAPISchema) -> OAPISchema:
        paths = spec.get("paths")
        if not isinstance(paths, dict):
            return spec

        for item in paths.values():
            assert isinstance(item, dict)
            self._convert_path_item(item)

        return spec

    OPERATIONS = (
        "get",
        "put",
        "post",
        "delete",
        "options",
        "head",
        "patch",
        "trace",
    )

    def _convert_path_item(self, item: dict) -> None:
        self._convert_parameters(item.get("parameters"))
        for k in self.OPERATIONS:
            if operation := item.get(k):
                self._convert_operation(operation)

    def _convert_operation(self, op: dict) -> None:
        self._convert_parameters(op.get("parameters"))

    def _convert_parameters(self, params: Any) -> None:
        if not isinstance(params, list):
            return
        for p in params:
            self._convert_parameter(p)

    def _convert_parameter(self, p: Any) -> None:
        if not isinstance(p, dict):
            return
        if p.get("$ref"):
            return
        schema = p.setdefault("schema", {})
        self._convert_parameter_schema(schema)

    def _convert_parameter_schema(self, schema: Any) -> None:
        assert isinstance(schema, dict)
        if schema.get("$ref"):
            return
        t = schema.setdefault("type", "string")
        if t == "array":
            items = schema.setdefault("items", {})
            self._convert_parameter_schema(items)
