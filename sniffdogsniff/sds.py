import argparse
import json
from urllib.parse import urlparse
import tqdm
import pandas as pd
from requests_html import HTMLSession


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
            # print(engine['name'], c)
            search_url = c.xpath(engine['result_url_filter'], first=True)
            search_title = c.xpath(engine['result_title_filter'], first=True)

            if search_url is not None:
                search_url = search_url.replace(' ', '')
            parsed_url = urlparse(search_url)

            if parsed_url.scheme in ['http', 'https']:
                df = df.append({'engine': engine['name'], 'title': search_title, 'search_url': search_url},
                               ignore_index=True)

    return df


def get_video_searches(search_query: str, config: dict) -> pd.DataFrame:
    df = get_dataframe()
    for engine in config['video_engines']:
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

    return df


def get_dataframe():
    df = pd.DataFrame(columns=['engine', 'title', 'search_url'])
    return df


def merge_frames(frame1: pd.DataFrame, frame2: pd.DataFrame) -> pd.DataFrame:
    return pd.concat([frame1, frame2], ignore_index=True)


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
    parser.add_argument('-u', '--unify', action='store_true',
                        help='use it if you want an output without duplicates, and not grouped by engine')
    parser.add_argument('-t', '--type', type=str,
                        help='use it if you want to do different types of search -t [NORMAL,VIDEO,SOCIAL]'
                             'can be more than one separated by comma ex: -t VIDEO,SOCIAL ,default is NORMAL',
                        default='NORMAL')
    parser.add_argument('-c', '--compare',
                        help='use it if you want to compare two or more searches by specific operand')
    return parser


def perform_searches(search_queries: str, config, number, wanted_searches):
    multiple_search_dataframes = list()
    queries = search_queries.split(',')
    for index in tqdm.tqdm(range(0, len(queries)), desc='Searching queries...'):
        sdf = get_dataframe()
        query = queries[index]
        if 'NORMAL' in wanted_searches:
            sdf = merge_frames(sdf,
                               get_searches(query, config, number))
        if 'VIDEO' in wanted_searches:
            sdf = merge_frames(sdf,
                               get_video_searches(query, config))

        multiple_search_dataframes.append(sdf)

    return multiple_search_dataframes


def parse_compare_arguments(compare_arg: str, search_dataframes: list, search_queries):
    if compare_arg.lower() == 'count':
        sq = search_queries.split(',')
        for elem in range(0, len(search_dataframes)):
            print(sq[elem], '\t::\t\t', str(len(search_dataframes[elem])))

        return search_dataframes


def main():
    args = get_parser().parse_args()
    config = get_config('engines.json')

    wanted_searches = args.type.split(',')

    search_dataframes = perform_searches(args.search_query, config, args.number, wanted_searches)
    out_dataframes = list()

    if args.unify:
        unified_dataframe = None
        for df in search_dataframes:
            unified_dataframe = merge_frames(unified_dataframe, get_unified_searches(df))

        out_dataframes.append(unified_dataframe)

    else:
        out_dataframes = search_dataframes

    if args.verbose:
        for df in search_dataframes:
            print(df)

    if args.compare is not None:
        print(len(search_dataframes))
        parse_compare_arguments(args.compare, search_dataframes, args.search_query)

    if args.output is not None:
        if len(out_dataframes) <= 1:
            if args.format == 'CSV':
                df.to_csv(args.output)
            else:
                export_df_html(df, args.output)
        else:
            for index in range(0, len(out_dataframes)):
                df = search_dataframes[index]

                if args.format == 'CSV':
                    df.to_csv(str(args.output + str(index)))
                else:
                    export_df_html(df, args.output)


if __name__ == '__main__':
    main()
