from spec_types import OAPISchema, SpecTranformer
from typing import List

# TODO: Add other operation methods?
read_requirements = ["get"]
write_requirements = ["post", "patch", "delete"]


class AddSecurityTransformer(SpecTranformer):
    def __init__(self, product_name: str):
        self.product_name = product_name

    def get_security_schema(self, http_method: str) -> List[OAPISchema] | None:
        scope = ""
        if http_method in read_requirements:
            scope = "read"
        elif http_method in write_requirements:
            scope = "write"
        else:
            return None

        return [{"OAuth2": [self.product_name + "." + scope]}]

    def transform(self, spec: OAPISchema) -> OAPISchema:
        """
        Assume all operations need security and add them, with scope using
        product_name
        """

        paths = spec.get("paths", {})
        for operations in paths.values():
            for http_method, op in operations.items():
                if op["security"] is not None:
                    continue

                security = self.get_security_schema(http_method)
                if security is None:
                    continue

                op["security"] = security

        return spec
