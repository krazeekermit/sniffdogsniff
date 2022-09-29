from sds.sdsconfigs import SdsConfigs
import logging
from sds.node import NodeManager, start_sds_node

if __name__ == '__main__': # standalone node start for testing only
    logging.basicConfig(level=logging.DEBUG)
    configs = SdsConfigs()
    configs.read_from_file('config.ini')
    manager = NodeManager(configs)
    start_sds_node(manager)
