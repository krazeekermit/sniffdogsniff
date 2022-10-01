import json


class JsonSerializer:
    def __init__(self):
        self._classes_encoding_functions = dict()
        self._classes_attributes = dict()
        self._classes_decode_functions = dict()

    """ Assumes that the object implements the __dict__ method """
    @staticmethod
    def _raw_serializer(o):
        return o.__dict__()

    """ Assumes that class implements has kwargs constructor """
    @staticmethod
    def _raw_deserializer(klass: type, obj_dict: dict):
        instance = klass(**obj_dict)
        return instance

    def register_object_custom(self, klass: type, object_attributes: list, encode_fun: callable, decode_fun: callable):
        self._classes_attributes[klass] = object_attributes
        self._classes_encoding_functions[klass] = encode_fun
        self._classes_decode_functions[klass] = decode_fun

    def register_object(self, klass: type, json_object_attributes: list):
        self.register_object_custom(klass, json_object_attributes, self._raw_serializer, self._raw_deserializer)

    def _default(self, o):
        for klass, fun in self._classes_encoding_functions.items():
            if isinstance(o, klass):
                return fun(o)
        return o

    def dumps(self, s):
        return json.dumps(s, default=self._default)

    def _object_hook(self, dictionary: dict):
        for klass, attrs in self._classes_attributes.items():
            if all(item in attrs for item in dictionary.keys()) and len(attrs) == len(dictionary.keys()):
                return self._classes_decode_functions[klass](klass, dictionary)
        return dictionary

    def loads(self, s):
        return json.loads(s, object_hook=self._object_hook)


