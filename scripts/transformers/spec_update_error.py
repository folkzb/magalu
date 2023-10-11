from spec_types import OAPISchema, SpecTranformer


class UpdateErrorTransformer(SpecTranformer):
    def transform(self, spec: OAPISchema) -> OAPISchema:
        """
        Kong modifies the error messages. Instead of the default object with details
        key with an array of items, it simplifies the error response with an object
        containing `message` and `slug`:

        Internal Error:
        {
            "detail": [
                "loc": ["string", 1]
                "msg": "foo",
                "type":  "bar"
            ]
        }

        Kong Error:
        {
            "message": "foo",
            "slug": "bar
        }

        This function patches any component in the schema markes as error and replace
        with `message` and `slug` object definition
        """
        components_schema = spec.get("components", {}).get("schemas", {})
        for coponent_name, schema in components_schema.items():
            if "error" not in coponent_name.lower():
                continue
            schema["type"] = "object"
            schema["properties"] = {
                "message": {"title": "Message", "type": "string"},
                "slug": {"title": "Slug", "type": "string"},
            }
            schema["example"] = {"message": "Unauthorized", "slug": "Unauthorized"}

        return spec
