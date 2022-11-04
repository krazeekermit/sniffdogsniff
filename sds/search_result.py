import hashlib
import json


class SearchResult:

    def __init__(self, **kwargs):
        self._hash = kwargs.get('hash', '')
        self._title = kwargs['title']
        self._url = kwargs['url']
        self._description = kwargs['description']
        self._content_type = kwargs.get('content_type', 'text/html')
        self._score = kwargs.get('score', 0)
        self._auto_hash()

    def calculate_hash(self):
        """
        Merkle tree style hashing of search result
        :return: the sha256 hash as string
        """
        to_hash = ''
        for value in [self._url, self._title, self._description, self._content_type]:
            to_hash += hashlib.sha256(value.encode()).hexdigest()
        return hashlib.sha256(to_hash.encode()).hexdigest()

    def _auto_hash(self):
        self._hash = self.calculate_hash()

    def __dict__(self):
        return {
            'hash': self._hash,
            'title': self._title,
            'url': self._url,
            'description': self._description,
            'content_type': self._content_type,
            'score': self._score
        }

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

    @property
    def content_type(self):
        return self._content_type

    def update_score(self, score: int):
        if score > self._score:
            self._score = score

    @property
    def score(self):
        return self._score


if __name__ == '__main__':
    sr = SearchResult(title='Title', url='http://www.google.com/', description='The world worst search engine')
    print(sr.hash)
