from sds.node import NodeManager
from flask import Flask, render_template, redirect, url_for
from flask.globals import request


class SdsWebApp:
    def __init__(self, node: NodeManager):
        self._node = node
        self._app = Flask(__name__)
        self._app.jinja_env.globals.update(content_type_img_path=self.content_type_img_path)
        self._app.add_url_rule('/', view_func=self.search_home)
        self._app.add_url_rule('/search', view_func=self.do_search)
        self._app.add_url_rule('/insert_link', view_func=self.search_home, methods=['GET', 'POST'])

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

    def do_insert_link(self):
        if request.method == 'POST':
            self._node.insert_new_search_result(
                request.form['link_title'], request.form['link_url'], request.form['link_description'], 'text/html'
            )
            return "<html><h1>Link insertion successful!</h1></html>"
        else:
            return render_template('insert_link.html')

    def start_ui(self, address, port):
        self._app.run(address, port)
