#!/usr/bin/env python3
import logging
import sys

from sds.configs import NodeConfigurations
from argparse import ArgumentParser
from sds.node import LocalNode
from sds.node_sync import NodeSyncServer, NodeSyncWorker
from webui.webui import SdsWebService


def parse_args():
    parser = ArgumentParser()
    parser.add_argument('-c', '--configfile', type=str, default='./config.ini',
                        help='Define config file path')
    return parser.parse_args()


def start_node():
    local_node = LocalNode(configs)
    server = NodeSyncServer(local_node)
    sync_client = NodeSyncWorker(local_node)

    app = SdsWebService(local_node)
    try:
        server.start()
        sync_client.start()
        app.start_web_service(configs.web_service_http_host, configs.web_service_http_port)
        logging.info('SniffDogSniff started, press CTRL+C to stop...')
    except KeyboardInterrupt:
        pass
    finally:
        logging.info('Awaiting SniffDogSniff to stop...')
        sync_client.stop_client()
        sync_client.join()
        server.stop_server()
        server.join()


if __name__ == '__main__':
    arguments = parse_args()
    configs = NodeConfigurations()
    configs.read_from_file(arguments.configfile)
    logging.basicConfig(level=configs.log_level, format='%(asctime)s %(levelname)8s %(name)15s - %(message)s')

    start_node()
    sys.exit(0)
