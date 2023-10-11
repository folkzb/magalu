from typing import Any, Dict, Protocol

OAPISchema = Dict[str, Any]


class SpecTranformer(Protocol):
    def transform(self, spec: OAPISchema) -> OAPISchema:
        pass
