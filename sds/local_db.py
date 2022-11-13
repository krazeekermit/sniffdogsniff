import base64
from os.path import exists
import sqlite3
from sds.search_result import SearchResult


class LocalResultsDB:

    def __init__(self, db_path):
        self._db_path = db_path
        self._conn = None
        self._check_db_exists()

    def _check_db_exists(self):
        if not exists(self._db_path):
            self._conn = sqlite3.connect(self._db_path, check_same_thread=False)
            self._conn.execute(
                'create table search_cache(hash text, title text, search_url text, description text,'
                ' content_type text, score int)'
            )
            self._conn.commit()
        else:
            self._conn = sqlite3.connect(self._db_path, check_same_thread=False)

    def _insert_records_and_commit(self, results: list):
        # self._conn.execute('delete from search_cache')
        for sr in results:
            self._conn.execute(
                f'insert or ignore into search_cache values("{base64.urlsafe_b64encode(sr.hash).decode()}",'
                f' "{sr.title}", "{sr.url}", "{sr.description}", "{sr.content_type}", {sr.score})'
            )

        self._conn.commit()

    def _do_query(self, sql_query: str) -> dict:
        cur = self._conn.cursor()
        cur.execute(sql_query)
        searches = []
        for row in cur.fetchall():
            searches.append(SearchResult(hash=base64.urlsafe_b64decode(row[0]), title=row[1], url=row[2],
                                         description=row[3], content_type=row[4], score=row[5]))

        return searches

    def search(self, query) -> dict:
        search_results = self._do_query(self._build_sql_query(query))
        print(f'results lenght {len(search_results)}')
        return search_results

    def get_searches(self) -> list:
        return self._do_query('select * from search_cache')

    def get_hashes(self) -> list:
        hashes = []
        for sr in self.get_searches():
            hashes.append(sr.hash)
        return hashes

    def sync(self, new_searches: list):
        # for h, v in new_searches.items():
        #     if h in searches.keys():
        #         v.update_score(searches[h].score)
        #     searches[h] = v
        hashes_set = set(self.get_hashes())
        for sr in new_searches:
            if sr.hash in hashes_set:
                new_searches.remove(sr)
        self._insert_records_and_commit(new_searches)

    def sync_from(self, new_searches: list):
        valid_searches = list()
        for sr in new_searches:
            if sr.is_consistent():
                valid_searches.append(sr)
        self.sync(valid_searches)

    @staticmethod
    def _build_sql_query(query_text: str) -> str:
        txt = query_text.lower()
        print(txt)
        query = f'select * from search_cache where lower(description)' \
                f' like "%{txt}%" or lower(title) like "%{txt}%" or lower(search_url) like "%{txt}%"'
        # for kw in txt.split(' '):
        #     if kw.isnumeric():
        #         continue
        #     query += f' or lower(description) like "%{kw}%" or lower(title) like "%{kw}%"' \
        #              f' or lower(search_url) like "%{kw}%"'
        return query
