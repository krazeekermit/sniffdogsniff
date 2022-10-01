import pandas as pd
import json
import os

import requests
import time


def new_searches_df() -> pd.DataFrame:
    return pd.DataFrame(columns=['engine', 'title', 'search_url', 'description'])


def set_clickable_links(self, link):
    return f'<a href="{link}">{link}</a>'


def merge_frames(frame1: pd.DataFrame, frame2: pd.DataFrame) -> pd.DataFrame:
    return pd.concat([frame1, frame2], ignore_index=True)


def json_to_dict(filename: str):
    with open(filename) as fp:
        return json.load(fp)


def clean_string(text: str):
    if text is None:
        return ""
    else:
        return text.replace('"', '')


def find_suitable_string(results: list):
    for s in results:
        text = results.pop()
        if not (text == ""):
            if len(text) > 5:
                return text
