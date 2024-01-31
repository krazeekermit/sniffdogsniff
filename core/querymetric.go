package core

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/sniffdogsniff/kademlia"
)

var unusefulWords = []string{
	"a", "an", "is", "was", "where", "there", "that", "of", "or", "any",
}

func skip(word string) bool {
	for _, st := range unusefulWords {
		if word == st {
			return true
		}
	}
	return false
}

func ToQueryTokens(s string) []string {
	//trim duplicates spaces
	r := regexp.MustCompile(`\s+`)
	s = r.ReplaceAllString(s, " ")

	list := make([]string, 0)

	for _, w := range strings.Split(s, " ") {
		url, err := url.ParseRequestURI(w)
		if err != nil {
			w = strings.TrimLeft(w, ":;,.#(")
			w = strings.TrimRight(w, ":;,.#)")
			if skip(w) {
				continue
			}
		} else {
			w = url.Host
			dotCount := strings.Count(w, ".")
			if dotCount > 1 {
				//keeps the most significant part of the url if the url is foo.bar.com it keeps bar.com
				for i := 0; i < dotCount-1; i++ {
					w = w[strings.Index(w, ".")+1:]
				}
			}
		}
		list = append(list, w)
	}

	return list
}

/*
	First 1 to 4 words represents the 4 KadIds That are used to evaluate distance,
	since we are a search engine we do not have a precise key to find a value across
	the network and so we need to find a certain metric to distribute and then retrieve
	SearchResults across the network using some search query metric.

	The metric (both for results and users queries) is calculated using the first four
	non duplicated words of the query, each word id is "hashed" into a kademlia Id and
	then the hashed id are added into the metric list.
*/

func evalQueryMetric(query []string) []kademlia.KadId {
	//first pass split and drop duplicates

	uniqueWords := make([]string, 0)
	for _, w := range query {
		if len(uniqueWords) >= 5 {
			break
		}
		duplicate := false
		for _, iw := range uniqueWords {
			if w == iw {
				duplicate = true
				break
			}
		}
		if !duplicate {
			uniqueWords = append(uniqueWords, w)
		}
	}

	wordIds := make([]kademlia.KadId, len(uniqueWords))
	for i := 0; i < len(uniqueWords); i++ {
		wordIds[i] = kademlia.NewKadId(uniqueWords[i])
	}

	return wordIds
}

func EvalQueryMetrics(query string) []kademlia.KadId {
	queryLower := strings.ToLower(query)

	return evalQueryMetric(ToQueryTokens(queryLower))
}
