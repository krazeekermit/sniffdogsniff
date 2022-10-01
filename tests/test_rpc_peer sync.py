import time
import unittest
from unittest import TestCase
from unittest.mock import Mock, MagicMock
from sdsjsonrpc import connect


from sds.node import NodeManager
from sds.sdsconfigs import SdsConfigs
from sds.search_result import SearchResult
import utils_for_tests
import logging


def setup_server():
    logging.basicConfig(logging.DEBUG)
    configs = SdsConfigs()
    configs.read_from_file('./config.test.ini')
    node_manager = NodeManager(configs)
    node_manager.request_node_searches = MagicMock(return_value={
        'c1f537dbc72e4b72eeaeb4b61f33a068268da566b5c05f200bd21844cf534634': SearchResult(
            hash='c1f537dbc72e4b72eeaeb4b61f33a068268da566b5c05f200bd21844cf534634',
            title='title1',
            url='ul1',
            description='desc1'
        )
    })
    utils_for_tests.start_test_rpc_server(node_manager)

    time.sleep(3)


class TestRpcRequestsToServer(TestCase):

    def test_rpc_request_searches(self):
        client = connect('127.0.0.1', 28600)
        client.serializer.register_object(SearchResult, ['hash', 'title', 'url', 'description'])
        response = client.request_node_searches_db_data([])
        self.assertTrue('c1f537dbc72e4b72eeaeb4b61f33a068268da566b5c05f200bd21844cf534634' in response.keys())
        result_obj = response['c1f537dbc72e4b72eeaeb4b61f33a068268da566b5c05f200bd21844cf534634']
        self.assertIsInstance(result_obj, SearchResult)
        self.assertEqual(result_obj.hash, 'c1f537dbc72e4b72eeaeb4b61f33a068268da566b5c05f200bd21844cf534634')
        self.assertEqual(result_obj.title, 'title1')
        self.assertEqual(result_obj.url, 'http://search1.com/')
        self.assertEqual(result_obj.description, 'blablabla1')

    def test_rpc_request_peers(self):
        client = connect('127.0.0.1', 28600)
        client.serializer.register_object(SearchResult, ['hash', 'title', 'url', 'description'])
        response = client.request_node_searches_db_data([])
        self.assertTrue('c1f537dbc72e4b72eeaeb4b61f33a068268da566b5c05f200bd21844cf534634' in response.keys())
        result_obj = response['c1f537dbc72e4b72eeaeb4b61f33a068268da566b5c05f200bd21844cf534634']
        self.assertIsInstance(result_obj, SearchResult)
        self.assertEqual(result_obj.hash, 'c1f537dbc72e4b72eeaeb4b61f33a068268da566b5c05f200bd21844cf534634')
        self.assertEqual(result_obj.title, 'title1')
        self.assertEqual(result_obj.url, 'http://search1.com/')
        self.assertEqual(result_obj.description, 'blablabla1')


if __name__ == '__main__':
    setup_server()
    unittest.main()




