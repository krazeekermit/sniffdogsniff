from sdsrpc import request_code


class RequestDispatcher:
    def __init__(self):
        self._registered_functions = {}

    def register_function(self, func_code: int, func: callable):
        self._registered_functions[func_code] = func

    def dispatch(self, func_code: int, args) -> tuple:
        try:
            return request_code.RETURN_CODE, func_code, self._registered_functions[func_code](*args)
        except TypeError:
            return request_code.ERROR_CODE, func_code, f'Wrong arguments for function {func_code}'
        except KeyError:
            return request_code.ERROR_CODE, func_code, f'Function {func_code} does not exists'
