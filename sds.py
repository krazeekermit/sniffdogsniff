from requests_html import HTMLSession
import pandas as pd
from urllib.parse import urlparse
import json
import argparse


def get_config(fname) -> dict:
    with open(fname) as fi:
        return json.load(fi)


def set_clickable_links(link):
    return f'<a href="{link}">{link}</a>'


def get_unified_searches(df: pd.DataFrame) -> pd.DataFrame:
    df.pop('engine')
    df = pd.DataFrame.drop_duplicates(df)
    return df


def export_df_html(df: pd.DataFrame, output_file):
    df['search_url'] = df['search_url'].apply(set_clickable_links)
    df.to_html(output_file, render_links=True, escape=False)


def get_searches(search_query: str, config: dict, nr: int) -> pd.DataFrame:
    df = get_dataframe()
    for engine in config['engines']:
        session = HTMLSession()
        r = session.get(engine['search_url'] + search_query +
                        engine['number_results_arg'] + str(nr))

        for c in r.html.find(engine['result_container_filter']):
            search_url = c.xpath(engine['result_url_filter'], first=True)
            search_title = c.xpath(engine['result_title_filter'], first=True)
            parsed_url = urlparse(search_url)
            # print(engine['name'], search_url, search_title)
            if parsed_url.scheme in ['http', 'https']:
                df = df.append({'engine': engine['name'], 'title': search_title, 'search_url': search_url},
                               ignore_index=True)

    return df


def get_dataframe():
    df = pd.DataFrame(columns=['engine', 'title', 'search_url'])
    return df


def get_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description='SniffDogSniff is a Web Scraping automated searching tool!!!')
    parser.add_argument('search_query', help='String or something you want to search', type=str)
    parser.add_argument('-v', '--verbose', action='store_true', help='Use this if you want to see a verbose output')
    parser.add_argument('output', help='the output file (see format)', type=str)
    parser.add_argument('-f', '--format', type=str,
                        help='is used to decide in which format you want to save the search. '
                             'Default is CSV, -f [CSV, HTML]', default='CSV')
    parser.add_argument('-n', '--number', type=int,
                        help='is used to decide number of results asked to engines. '
                             'Default is 10, -n 10', default=10)
    parser.add_argument('-u', '--unified', action='store_true',
                        help='use it if you want an output without duplicates, and not grouped by engine')
    return parser


def main():
    args = get_parser().parse_args()
    config = get_config('engines.json')
    sdf = get_searches(args.search_query, config, args.number)
    if args.unified:
        sdf = get_unified_searches(sdf)

    if args.verbose:
        print(sdf)

    if args.output is not None:
        if args.format == 'CSV':
            sdf.to_csv(args.output)
        else:
            export_df_html(sdf, args.output)


if __name__ == '__main__':
    main()
