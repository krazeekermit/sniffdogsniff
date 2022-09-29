"""
JSONRPCTCP Library default imports
"""

__all__ = ['config', 'history', 'connect', 'logger', 'start_server']

# Set up the basic logging system for JSONRPCTCP.
import logging


class NullLogHandler(logging.Handler):
    """ Ensures that other libraries don't see 'No handler...' output """

    def emit(self, record):
        pass


logger = logging.getLogger('JSONRPCTCP')
logger.addHandler(NullLogHandler())

# Default imports
from sdsjsonrpc.config import Config

config = Config.instance()
from sdsjsonrpc.history import History

history = History.instance()
from sdsjsonrpc.client import connect
from sdsjsonrpc.server import start_server
