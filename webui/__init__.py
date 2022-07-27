from flask import Flask, render_template
from flask.globals import request
from sds import sdsutils
from sds.sds import SniffingDog

engines = {
    "engines": [
        {
            "name": "Google",
            "headers": {"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/70.0.3538.77 Safari/537.36"},
            "search_url": "https://www.google.com/search?q=",
            "result_container_filter": "div.g",
            "result_url_filter": "//a/@href",
            "result_title_filter": "//h3/text()",
            "result_description_filter": "//span/text()",
            "number_results_arg": "&num="
        },
        {
            "name": "Bing",
            "headers": {"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/70.0.3538.77 Safari/537.36"},
            "search_url": "https://www.bing.com/search?q=",
            "result_container_filter": "li.b_algo",
            "result_url_filter": "//a/@href",
            "result_title_filter": "//a/text()",
            "result_description_filter": "//div[@class='b_snippet']//p/text()",
            "number_results_arg": "&count="
        }
    ],
    "video_engines": [
        {
            "name": "Rumble",
            "search_url": "https://rumble.com/search/video?q=",
            "result_container_filter": "article.video-item",
            "result_url_filter": "//a/@href",
            "result_title_filter": "//h3/text()",
            "url_prefix": "https://www.rumble.com"
        }
    ]
}

app = Flask(__name__)
sniffer = SniffingDog(engines)


@app.route('/')
def search_home():
    return render_template('home.html')


@app.route('/search')
def do_search():
    if request.method == 'GET':
        sniffer.do_search(request.args.get('q'), 1000)
        sniffer.unify_searches()
        searches = sniffer.get_searches_as_dicts
        return render_template('results.html', searches=searches)
