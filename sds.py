from requests_html import HTMLSession
import pandas as pd
from urllib.parse import urlparse
import json
import argparse


def get_config(fname) -> dict:
    with open(fname) as fi:
        return json.load(fi)


def get_searches(search_query: str, config: dict) -> pd.DataFrame:
    df = get_dataframe()
    for engine in config['engines']:
        session = HTMLSession()
        r = session.get(engine['search_url'] + search_query)
        # print(engine['name'], r.text)
        for c in r.html.find(engine['result_container_filter']):
            search_url = c.xpath(engine['result_url_filter'], first=True)
            search_title = c.xpath(engine['result_title_filter'], first=True)
            parsed_url = urlparse(search_url)
            # print(engine['name'], search_url, search_title)
            if parsed_url.scheme in ['http', 'https']:
                df = df.append({'engine': engine['name'], 'search_url': search_url, 'title': search_title},
                               ignore_index=True)

    return df


def get_dataframe():
    df = pd.DataFrame(columns=['engine', 'search_url', 'title'])
    return df


def get_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser()
    parser.add_argument('search_query', help='String or something you want to search', type=str)
    parser.add_argument('-v','--verbose', action='store_true', help='Use this if you want to see a verbose output')
    parser.add_argument('-o', '--output', help='Use this if you want to save in a csv file', type=str)
    return parser


def main():
    args = get_parser().parse_args()
    config = get_config('engines.json')
    sdf = get_searches(args.search_query, config)
    if args.output is not None:
        sdf.to_csv(args.output)

    print(sdf)


if __name__ == '__main__':
    main()
