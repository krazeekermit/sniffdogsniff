from flask import Flask, render_template, redirect
from flask.globals import request
from sds.node import NodeManager, start_sds_node
from sds.sdsconfigs import SdsConfigs
import logging
import os

app = Flask(__name__)


@app.route('/')
def search_home():
    return render_template('home.html')


@app.route('/search')
def do_search():
    if request.method == 'GET':
        searches = node.sniffing_dog.do_search(request.args.get('q'))
        return render_template('results.html', searches=searches)


@app.route('/insert_link', methods=['GET', 'POST'])
def do_insert_link():
    if request.method == 'POST':
        node.insert_new_search_result(
            request.form['link_title'], request.form['link_url'], request.form['link_description']
        )
        return "<html><h1>Link insertion successful!</h1></html>"
    else:
        return render_template('insert_link.html')


if __name__ == '__main__':
    logging.basicConfig(level=logging.DEBUG)
    configs = SdsConfigs()
    configs.read_from_file('../config.ini')
    node = NodeManager(configs)
    start_sds_node(node_manager=node)
    app.run('127.0.0.1', 5000)
