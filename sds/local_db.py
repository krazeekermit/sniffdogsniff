from os.path import exists
import sqlite3
import pandas as pd
from . import sdsutils


class LocalSearchDatabase:

    def __init__(self, db_path):
        self._db_path = db_path
        self._conn = None
        self._check_db_exists()

    def _check_db_exists(self):
        if not exists(self._db_path):
            self._conn = sqlite3.connect(self._db_path, check_same_thread=False)
            self._conn.execute(
                'create table search_cache(engine text, title text, search_url text, description text)')
            # self._conn.commit()
        else:
            self._conn = sqlite3.connect(self._db_path, check_same_thread=False)

    def _insert(self, df: pd.DataFrame):
        self._conn.execute('delete from search_cache')
        for index, row in df.iterrows():
            engine = row['engine']
            title = row['title']
            search_url = row['search_url']
            description = row['description']
            self._conn.execute(
                f'insert into search_cache values("{engine}", "{title}", "{search_url}", "{description}")'
            )

        self._conn.commit()

    def search(self, query: str) -> pd.DataFrame:
        return pd.read_sql_query(
            f'select * from search_cache where description like "%{query}%" or title like "%{query}%" or search_url like "%{query}%"',
            self._conn
        )

    def sync(self, df: pd.DataFrame):
        df_db = pd.read_sql_query(f'select * from search_cache', self._conn)
        self._insert(sdsutils.merge_frames(df, df_db))

    