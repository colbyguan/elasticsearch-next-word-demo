package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/olivere/elastic/v7"
)

type Server struct {
	client *elastic.Client
}

func NewServer() *Server {
	client, err := elastic.NewClient()
	Check(err)
	return &Server{
		client: client,
	}
}

func (s *Server) Start() {
	s.setupScripts()
	http.Handle("/", http.FileServer(http.Dir("./ui")))
	http.HandleFunc("/search", s.searchHandler)
	log.Panic(http.ListenAndServe(":8000", nil))
}

func (s *Server) setupScripts() {
	firstWordSource := "String val = doc[params.field].value; int sepIdx = val.indexOf(params.separator); return sepIdx > 0 ? val.substring(0,sepIdx) : val"
	nextWordSource := "String val = doc[params.field].value; int sepIdx = val.indexOf(params.separator, params.wordStart); return sepIdx > 0 ? val.substring(params.wordStart, sepIdx): val.substring(Integer.min(params.wordStart, val.length()));"
	req := elastic.NewPutScriptService(s.client)
	for name, source := range map[string]string{"first-word": firstWordSource, "next-word": nextWordSource} {
		body := map[string]map[string]string{
			"script": {
				"lang":   "painless",
				"source": source,
			},
		}
		resp, err := req.Id(name).BodyJson(body).Do(context.Background())
		Check(err)
		fmt.Printf("put script %s: %v\n", name, resp.Acknowledged)
	}
}

type PrefixBody struct {
	Prefix      string `json:"prefix"`
	UseNextWord bool   `json:"useNextWord"`
}

func (s *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var prefixBody PrefixBody
	err := decoder.Decode(&prefixBody)
	if err != nil {
		fmt.Println(err)
		fmt.Fprintf(w, "%v", err)
		return
	}
	hits := s.doNextWordSearch(prefixBody.Prefix, prefixBody.UseNextWord)
	json.NewEncoder(w).Encode(hits)
}

func (s *Server) doNextWordSearch(prefix string, useNextWord bool) []string {
	q := elastic.NewPrefixQuery("search", prefix)
	search := s.client.Search().Index(INDEX).Query(q).Pretty(true)
	if useNextWord {
		scriptParams := map[string]interface{}{
			"field":     "search",
			"separator": SEPARATOR_STR,
		}
		var agg *elastic.TermsAggregation
		if strings.Contains(prefix, SEPARATOR_STR) {
			var wordStart int
			if prefix[len(prefix)-1] == SEPARATOR_RUNE {
				wordStart = len(prefix)
			} else {
				wordStart = strings.LastIndex(prefix, SEPARATOR_STR) + 1
			}
			scriptParams["wordStart"] = wordStart
			agg = elastic.NewTermsAggregation().Script(elastic.NewScriptStored("next-word").Params(scriptParams))
		} else {
			agg = elastic.NewTermsAggregation().Script(elastic.NewScriptStored("first-word").Params(scriptParams))
		}
		search = search.Size(0).Aggregation("uniques", agg)
		result, err := search.Do(context.Background())
		if err != nil {
			fmt.Println(err)
			return []string{}
		}
		aggResult, found := result.Aggregations.Terms("uniques")
		hits := []string{}
		if found {
			for _, bucket := range aggResult.Buckets {
				hits = append(hits, bucket.Key.(string))
			}
		}
		return hits
	} else {
		// default return limit (Size) is small, use a larger one
		result, err := search.Size(1000).Do(context.Background())
		if err != nil {
			fmt.Println(err)
			return []string{}
		}
		hits := []string{}
		for _, hit := range result.Hits.Hits {
			var doc SearchDoc
			err := json.Unmarshal(hit.Source, &doc)
			if err != nil {
				fmt.Println(err)
				return []string{}
			}
			hits = append(hits, doc.Search)
		}
		return hits
	}
}
