import json
from urllib.parse import urlparse
import socks


def json_to_dict(filename: str):
    with open(filename) as fp:
        return json.load(fp)


def clean_string(text: str):
    if text is None:
        return ""
    else:
        return text.replace('"', '')


def find_suitable_string(results: list):
    for s in results:
        text = results.pop()
        if not (text == ""):
            if len(text) > 5:
                return text


def string_to_host_port_tuple(addr: str):
    parsed_url = urlparse(addr)
    return parsed_url.hostname, parsed_url.port


def string_to_proxy_type(proxy_type: str):
    return socks.PROXY_TYPES.get(proxy_type.upper())


def content_type_to_mime_type(content_type: str):
    return content_type.split(';')[0].strip()


if __name__ == '__main__':
    print()




