
class RpcRequestException(Exception):
    def __init__(self, payload: str, fun_code: int):
        self._payload = payload
        self._fun_code = fun_code

    def __str__(self):
        return self._payload

    @property
    def function_code(self):
        return self._fun_code
