from urllib.parse import urlparse
import pandas as pd
from requests_html import HTMLSession
from . import sdsutils
from . import local_db


class SniffingDog:

    def __init__(self, engines_properties: dict):
        self._df = None
        self._reset_df()
        self._engines_properties = engines_properties
        self._local_db = local_db.LocalSearchDatabase('./search_cache.db')

    def get_unified_searches(self) -> pd.DataFrame:
        self._df.pop('engine')
        df = pd.DataFrame.drop_duplicates(self._df)
        return df

    def unify_searches(self) -> pd.DataFrame:
        self._df.pop('engine')
        df = pd.DataFrame.drop_duplicates(self._df)
        self._df = df

    def export_to_html(self, output_file):
        self._df['search_url'] = self._df['search_url'].apply(sdsutils.set_clickable_links)
        self._df.to_html(output_file, render_links=True, escape=False)

    def export_to_csv(self, output_file):
        self._df.to_csv(output_file)

    def do_search(self, search_query: str, nr: int) -> pd.DataFrame:
        df = self._local_db.search(search_query)
        self._reset_df()
        for engine in self._engines_properties['engines']:
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

        self._local_db.sync(df)
        self._df = df

    def do_video_search(self, search_query: str) -> pd.DataFrame:
        self._reset_df()
        for engine in self._engines_properties['video_engines']:
            session = HTMLSession()
            r = session.get(engine['search_url'] + search_query)

            for c in r.html.find(engine['result_container_filter']):
                search_url = c.xpath(engine['result_url_filter'], first=True)
                search_title = c.xpath(engine['result_title_filter'], first=True)

                if search_url is not None:
                    search_url = search_url.replace(' ', '')

                decent_url = engine['url_prefix'] + search_url
                parsed_url = urlparse(decent_url)

                if parsed_url.scheme in ['http', 'https']:
                    df = df.append({'engine': engine['name'], 'title': search_title, 'search_url': decent_url},
                                   ignore_index=True)

        self._df = df

    @property
    def get_searches(self):
        return self._df

    @property
    def get_searches_as_dicts(self) -> list:
        return self._df.to_dict('records')

    def _reset_df(self):
        self._df = sdsutils.new_searches_df()
