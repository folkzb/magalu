from typing import Protocol
from oapi_types import OAPI


class SpecTranformer(Protocol):
    def transform(self, oapi: OAPI):
        pass
