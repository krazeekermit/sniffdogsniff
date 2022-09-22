import hashlib


class SearchResult:

    def __init__(self, *args):
        self._hash = args[0]
        self._title = args[1]
        self._url = args[2]
        self._description = args[3]
        self._args = args
        self._auto_hash()

    def calculate_hash(self):
        to_hash = self._url + self._title
        return hashlib.sha256(to_hash.encode()).hexdigest()

    def _auto_hash(self):
        self._hash = self.calculate_hash()

    def as_dict(self) -> dict:
        return {'title': self._title, 'search_url': self._url, 'description': self._description}

    def _serialize(self):
        return (self._args,
                {'hash': self._hash, 'title': self._title, 'url': self._url, 'description': self._description}
                )

    def is_consistent(self) -> bool:
        return self._hash == self.calculate_hash()

    @property
    def hash(self):
        return self._hash

    @hash.setter
    def hash(self, h):
        self._hash = h

    @property
    def title(self):
        return self._title

    @title.setter
    def title(self, tit):
        self._title = tit

    @property
    def url(self):
        return self._url

    @url.setter
    def url(self, search_url):
        self._url = search_url

    @property
    def description(self):
        return self._description

    @description.setter
    def description(self, desc):
        self._description = desc
