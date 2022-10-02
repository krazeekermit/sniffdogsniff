from sds.configs import NodeConfigurations
import logging
from sds.node import NodeManager, start_sds_node

if __name__ == '__main__': # standalone node start for testing only
    logging.basicConfig(level=logging.DEBUG)
    configs = NodeConfigurations()
    configs.read_from_file('config.ini')
    manager = NodeManager(configs)
    start_sds_node(manager)
