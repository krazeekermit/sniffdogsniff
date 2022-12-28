import logging

from sds.node import LocalNode
from werkzeug.serving import make_server
from flask import Flask, render_template, redirect, url_for
from flask.globals import request


class SdsWebService:
    def __init__(self, node: LocalNode):
        self._node = node
        self._logger = logging.getLogger('Web Service')
        self._app = Flask(__name__)
        self._app.jinja_env.globals.update(content_type_img_path=self.content_type_img_path)
        self._app.add_url_rule('/', view_func=self.search_home)
        self._app.add_url_rule('/search', view_func=self.do_search)
        self._app.add_url_rule('/redirect', view_func=self.go_to_link)
        self._app.add_url_rule('/insert_link', view_func=self.search_home, methods=['GET', 'POST'])
        self._server = None

        self._current_results = None

    @staticmethod
    def content_type_img_path(c: str):
        return url_for('static', filename=f'content_{c.split("/")[1]}.svg')

    @staticmethod
    def search_home():
        return render_template('home.html')

    def do_search(self):
        if request.method == 'GET':
            self._node.unlock()
            searches = self._node.search(request.args.get('q'))
            self._node.lock()
            return render_template('results.html', searches=searches)

    def go_to_link(self):
        if request.method == 'GET':
            self._node.update_result_score(bytes.fromhex(request.args.get('hash')), 1)
            return redirect(request.args.get('url'))

    def do_insert_link(self):
        if request.method == 'POST':
            self._node.insert_new_search_result(
                request.form['link_title'], request.form['link_url'], request.form['link_description'], 'text/html'
            )
            return "<html><h1>Link insertion successful!</h1></html>"
        else:
            return render_template('insert_link.html')

    def start_web_service(self, address, port):
        self._logger.info(f'Started web server on http://{address}:{port}')
        self._server = make_server(address, port, self._app)
        self._server.serve_forever()

    def stop_web_service(self):
        self._logger.info('Shutting down web service...')
        self._server.shutdown()
