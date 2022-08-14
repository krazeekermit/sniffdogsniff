import pandas as pd
from requests_html import HTMLSession
import requests
from urllib.parse import urlparse
from . import sdsutils


def search_from_engines(search_query: str, nr: int, engines: dict) -> pd.DataFrame:
    df = sdsutils.new_searches_df()
    for engine in engines:
        session = HTMLSession()
        r = session.get(engine['search_url'] + search_query +
                        engine['number_results_arg'] + str(nr),
                        headers=engine['headers'])

        for c in r.html.find(engine['result_container_filter']):
            search_url = c.xpath(engine['result_url_filter'], first=True)
            search_title = sdsutils.clean_string(c.xpath(engine['result_title_filter'], first=True))
            if search_url is not None:
                search_url = search_url.replace(' ', '')
            parsed_url = urlparse(search_url)

            if parsed_url.scheme in ['http', 'https']:
                try:
                    search_desc = sdsutils.clean_string(session.get(
                        search_url, headers=engine['headers'], timeout=0.75
                    ).html.xpath('//meta[@name="description"]/@content', first=True))
                except:
                    search_desc = ""

                df = df.append({'engine': engine['name'],
                                'title': search_title,
                                'search_url': search_url,
                                'description': search_desc}, ignore_index=True)
    return df


def search_from_peers(addrs: list, search_query: str):
    df = sdsutils.new_searches_df()
    for addr in addrs:
        try:
            resp = requests.post(f'{addr}/api/search', json={'query', search_query})
            if resp.content:
                df = df.append(pd.read_json(resp.content))
        except:
            print(f'{addr} error!')

    return df

