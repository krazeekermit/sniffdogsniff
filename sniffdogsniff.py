from sds.configs import NodeConfigurations
import logging
from sds.node import NodeManager, start_sds_node
from webui.webui import SdsWebApp


if __name__ == '__main__': # standalone node start for testing only
    configs = NodeConfigurations()
    configs.read_from_file('config.ini')
    logging.basicConfig(level=configs.log_level, format='%(asctime)s %(levelname)8s %(name)15s - %(message)s')
    manager = NodeManager(configs)
    start_sds_node(manager)
    app = SdsWebApp(node=manager)
    app.start_ui(configs.web_service_http_host, configs.web_service_http_port)
