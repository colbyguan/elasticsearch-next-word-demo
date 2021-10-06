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
	firstWordSource := "String val = doc[params.field].value; int i = val.indexOf(params.separator); return i > 0 ? val.substring(0,i) : val"
	nextWordSource := "String val = doc[params.field].value; int i = val.indexOf(params.separator, params.prefixLength); return i > 0 ? val.substring(params.prefixLength, i): val.substring(Integer.min(params.prefixLength, val.length()));"
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
	Prefix string `json:"prefix"`
}

func (s *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var prefixBody PrefixBody
	err := decoder.Decode(&prefixBody)
	if err != nil {
		fmt.Fprintf(w, "%v", err)
		return
	}
	prefix := prefixBody.Prefix
	q := elastic.NewPrefixQuery("search", prefix)
	search := s.client.Search().Index(INDEX).Query(q).Pretty(true).Size(0)
	scriptParams := map[string]interface{}{
		"field":        "search",
		"separator":    " ",
		"prefixLength": len(prefix),
	}
	var agg *elastic.TermsAggregation
	if strings.Contains(prefix, " ") {
		var prefixLength int
		if prefix[len(prefix)-1] == ' ' {
			prefixLength = len(prefix)
		} else {
			prefixLength = strings.LastIndex(prefix, " ") + 1
		}
		scriptParams["prefixLength"] = prefixLength
		agg = elastic.NewTermsAggregation().Script(elastic.NewScriptStored("next-word").Params(scriptParams))
	} else {
		agg = elastic.NewTermsAggregation().Script(elastic.NewScriptStored("first-word").Params(scriptParams))
	}
	search = search.Aggregation("uniques", agg)
	result, err := search.Do(context.Background())
	if err != nil {
		fmt.Println(err)
	}
	aggResult, found := result.Aggregations.Terms("uniques")
	if !found {
		json.NewEncoder(w).Encode([]string{})
	} else {
		hits := []string{}
		for _, bucket := range aggResult.Buckets {
			hits = append(hits, bucket.Key.(string))
		}
		json.NewEncoder(w).Encode(hits)
	}
}
