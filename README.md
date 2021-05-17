# sniffDogSniff

Sniff Dog Sniff is a customizable multiple search web scraping tool 

### Install dependencies
```
$ pip3 install -r requirements.txt
```

### How to use
```
$ python3 sds.py [-h] [-v] [-o OUTPUT] search_query
```
* _search_query_ String or something you want to search
* -h, --help            show this help message and exit
*  -v, --verbose         Use this if you want to see a verbose output
* -o OUTPUT, --output OUTPUT Use this if you want to save in a csv file



### The config file (engines.json) (Advanced use)
```
{
  "engines": [
    {
      "name": "Google",
      "search_url": "https://www.google.com/search?q=",
      "result_container_filter": "div.g",
      "result_url_filter": "//a/@href",
      "result_title_filter": "//h3/text()"
    }
}
```
* _engines_ is a list of dictionary for each engine
* _name_ the search engine name
* _result_container_filter_ is the filter referring to the main result container uses html/css filtering
* _result_url_filter_ and _result_title_filter_ use both xpath filtering are used to determine respectively the search 
  url and the link text