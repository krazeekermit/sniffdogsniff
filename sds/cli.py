import argparse
import tqdm
from sds import SniffingDog
import sdsutils


def parse_compare_arguments(compare_arg: str, search_dataframes: list, search_queries):
    if compare_arg.lower() == 'count':
        sq = search_queries.split(',')
        for elem in range(0, len(search_dataframes)):
            print(sq[elem], '\t::\t\t', str(len(search_dataframes[elem])))

        return search_dataframes


def perform_searches(self, search_queries: str ,number: int, wanted_searches):

    for index in tqdm.tqdm(range(0, len(search_queries)), desc='Searching queries...'):
        query = search_queries[index]
        if 'NORMAL' in wanted_searches:
            sdf = self._merge_frames(sdf,
                                     sniffer.do_search(query, number))
        if 'VIDEO' in wanted_searches:
            sdf = self._merge_frames(sdf,
                                     sniffer.do_video_search(query))

    return sdf


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


def main():
    args = get_parser().parse_args()

    wanted_searches = args.type.split(',')

    search_dataframes = perform_searches(args.search_query.split(','), args.number, wanted_searches)
    out_dataframes = list()

    if args.unify:
        unified_dataframe = None
        for df in search_dataframes:
            unified_dataframe = sdsutils.merge_frames(unified_dataframe, sniffer.get_unified_searches())

        out_dataframes.append(unified_dataframe)

    else:
        out_dataframes = search_dataframes

    if args.verbose:
        for df in search_dataframes:
            print(df)

    if args.compare is not None:
        parse_compare_arguments(args.compare, search_dataframes, args.search_query)

    if args.output is not None:
        if args.format == 'CSV':
            sniffer.export_to_csv(args.output)
        else:
            sniffer.export_to_html(args.output)


if __name__ == '__main__':
    sniffer = SniffingDog(sdsutils.json_to_dict('engines.json'))
    main()