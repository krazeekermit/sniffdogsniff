from sdsrpc import request_code


class RequestDispatcher:
    def __init__(self):
        self._registered_functions = {}

    def register_function(self, func_code: int, func: callable):
        self._registered_functions[func_code] = func

    def dispatch(self, func_code: int, args):
        return self._registered_functions[func_code](*args)

