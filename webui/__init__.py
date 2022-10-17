from flask import Flask, render_template, redirect, url_for
from flask.globals import request
from sds.node import NodeManager, start_sds_node
from sds.configs import NodeConfigurations
import logging
import os

app = Flask(__name__)


def content_type_img_path(c: str):
    return url_for('static', filename=f'content_{c.split("/")[1]}.svg')


@app.route('/')
def search_home():
    return render_template('home.html')


@app.route('/search')
def do_search():
    if request.method == 'GET':
        node.unlock()
        searches = node.search(request.args.get('q'))
        node.lock()
        return render_template('results.html', searches=searches)


@app.route('/insert_link', methods=['GET', 'POST'])
def do_insert_link():
    if request.method == 'POST':
        node.insert_new_search_result(
            request.form['link_title'], request.form['link_url'], request.form['link_description'], 'text/html'
        )
        return "<html><h1>Link insertion successful!</h1></html>"
    else:
        return render_template('insert_link.html')


if __name__ == '__main__':
    app.jinja_env.globals.update(content_type_img_path=content_type_img_path)
    logging.basicConfig(level=logging.DEBUG)
    configs = NodeConfigurations()
    configs.read_from_file('../config.ini')
    node = NodeManager(configs)
    start_sds_node(node_manager=node)
    app.run('127.0.0.1', configs.web_service_http_port)
