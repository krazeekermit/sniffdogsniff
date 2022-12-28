import hashlib
import json


class SearchResult:

    def __init__(self, **kwargs):
        self._hash = kwargs.get('hash', None)
        self._title = kwargs['title']
        self._url = kwargs['url']
        self._description = kwargs['description']
        self._content_type = kwargs.get('content_type', 'text/html')
        self._score = kwargs.get('score', 0)
        if self._hash is None:
            self._hash = self.calculate_hash()

    def calculate_hash(self):
        """
        Merkle tree style hashing of search result
        :return: the sha256 hash as bytes
        """
        to_hash = b''
        for value in [self._url, self._title, self._description, self._content_type]:
            to_hash += hashlib.sha256(value.encode()).digest()
        return hashlib.sha256(to_hash).digest()

    def is_consistent(self) -> bool:
        return self._hash == self.calculate_hash()

    def __str__(self):
        return f'SearchResult: hash={self._hash.hex()}, title={self._title}, desc={self._description},' \
               f' mime={self._content_type}, score={self._score}'

    def __repr__(self):
        return self.__str__()

    def __hash__(self):
        return hash(self._hash)

    def __eq__(self, other):
        return self._hash == other.hash

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

    @property
    def content_type(self):
        return self._content_type

    def update_score(self, score: int):
        if score > self._score:
            self._score = score

    @property
    def score(self):
        return self._score

