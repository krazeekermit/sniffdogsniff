from sds.local_db import LocalResultsDB
from urllib.parse import urlparse
from requests_html import HTMLSession
import logging
from sds.search_result import SearchResult
from sds.utils import clean_string, content_type_to_mime_type


class SearchEngine:

    def __init__(self, name: str, query_url: str, result_container_filter: str, result_url_filter: str,
                 result_title_filter: str, user_agent: str):
        self._name = name
        self._query_url = query_url
        self._result_container_filter = result_container_filter
        self._result_url_filter = result_url_filter
        self._result_title_filter = result_title_filter
        self._http_headers = {"User-Agent": user_agent}

    def search(self, query: str):
        searches = []
        session = HTMLSession()
        r = session.get(self._query_url + query,
                        headers=self._http_headers)

        for c in r.html.find(self._result_container_filter):
            search_url = c.xpath(self._result_url_filter, first=True)
            search_title = clean_string(c.xpath(self._result_title_filter, first=True))
            if search_url is not None:
                search_url = search_url.replace(' ', '')
            parsed_url = urlparse(search_url)

            if parsed_url.scheme in ['http', 'https']:
                try:
                    resp = session.get(search_url, headers=self._http_headers, timeout=0.75)
                    content_type = content_type_to_mime_type(resp.headers['content-type'])
                    search_desc = clean_string(resp.html.xpath('//meta[@name="description"]/@content', first=True))
                except Exception as ex:
                    content_type = 'text/html'
                    search_desc = search_title

                result = SearchResult(title=search_title, url=search_url, description=search_desc,
                                      content_type=content_type)
                searches.append(result)
        return searches

    @property
    def name(self):
        return self._name


class SniffingDog:

    def __init__(self, engines: list, local_db: LocalResultsDB, minimum_search_results_threshold: int):
        self._engines = engines
        self._local_db = local_db
        self._minimum_search_results_thr = minimum_search_results_threshold

    def do_search(self, search_query: str, filter_content_types=[]) -> dict:
        searches = []
        searches.extend(self._local_db.search(search_query))

        if not len(searches) > self._minimum_search_results_thr:
            for e in self._engines:
                logging.debug(f'Searching results from {e.name}')
                try:
                    res = e.search(search_query)
                    searches.extend(res)
                except Exception as ex:
                    print(f'error in {e.name}= {ex.args}')

        self._local_db.sync(searches)

        # TODO Filter by content_type
        # to be implemented
        return searches
