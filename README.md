# sniffDogSniff

Sniff Dog Sniff is a customizable multiple search web scraping tool 

### Install dependencies
```
$ pip3 install -r requirements.txt
```

### How to use
```
$ python3 sds.py [-h] [-v] [-f FORMAT] [-n NUMBER] [-u] search_query output
```
* search_query          String or something you want to search (or multiple queries separated by comma)
* output                the output file (see format)

* -h, --help            show this help message and exit
* -v, --verbose         Use this if you want to see a verbose output
* -f FORMAT, --format FORMAT
                        is used to decide in which format you want to save the
                        search. Default is CSV, -f [CSV, HTML]
* -n NUMBER, --number NUMBER
                        is used to decide number of results asked to engines.
                        Default is 10, -n 10
* -u, --unified         use it if you want an output without duplicates, and
                        not grouped by engine
* -c OPERAND, --compare OPERAND
                        is used to compare different search results (see Compare)


### Compare
The flag -c is used to compare one or more queries search results with themselves (or the others if you 
use multiple search queries). How to use compare flag:
* -c COUNT or -c count      counts the number of results for each query and print the result

**work in progress**

### Graphical User Interface
To simplify life of users, i recently made a simple GUI with PyQt5. To run the Ui simply run:
```
$ python3 sds_gui.py
```


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