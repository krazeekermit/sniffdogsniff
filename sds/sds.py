from urllib.parse import urlparse
import pandas as pd
from requests_html import HTMLSession
from . import sdsutils
from . import local_db
from . import search_providers
from . import peers


class SniffingDog:

    def __init__(self, configs: dict):
        self._df = None
        self._reset_df()
        self._engines_properties = configs['search']['engines']
        self._local_db = local_db.LocalSearchDatabase(self._engines_properties['local_search_db_path'])
        self._sds_peers = peers.SdsPeers(configs)

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

    def do_search(self, search_query: str, nr: int, exclude_engines=False) -> pd.DataFrame:
        df = self._local_db.search(search_query)
        if not df.size > 50:
            df = df.append(
                search_providers.search_from_peers(self._sds_peers.get_peer_list, search_query),
                ignore_index=True
            )

        if not exclude_engines and not df.size > 50:
            df = df.append(
                search_providers.search_from_engines(search_query, nr, self._engines_properties),
                ignore_index=True
            )

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

    def _reset_df(self):
        self._df = sdsutils.new_searches_df()

    @property
    def get_searches(self):
        return self._df

    @property
    def get_searches_as_dicts(self) -> list:
        return self._df.to_dict('records')

    @property
    def peers(self) -> peers.SdsPeers:
        return self._sds_peers
