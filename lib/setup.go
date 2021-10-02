package lib

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/anaskhan96/soup"
	"github.com/olivere/elastic/v7"
)

func CreateTextFile() {
	searches := getSearchesFromLocalHtml()
	contents := []byte(strings.Join(searches, "\n"))
	os.WriteFile("./data.txt", contents, 0644)
}

func PopulateIndex() {
	data, err := os.ReadFile("./data.txt")
	Check(err)
	searches := strings.Split(string(data), "\n")
	client, err := elastic.NewClient()
	Check(err)

	exists, err := client.IndexExists(INDEX).Do(context.Background())
	Check(err)
	if exists {
		fmt.Println("index exists")
		deleteIndex, err := client.DeleteIndex(INDEX).Do(context.Background())
		Check(err)
		fmt.Printf("delete acknowledgement: %v\n", deleteIndex.Acknowledged)
	}

	bulkRequest := client.Bulk()
	for id, search := range searches {
		bulkRequest.Add(elastic.NewBulkIndexRequest().Index(INDEX).Type(TYPE).Id(strconv.Itoa(id)).Doc(SearchDoc{search}))
	}
	bulkResponse, err := bulkRequest.Do(context.Background())
	Check(err)
	indexed := bulkResponse.Indexed()
	fmt.Printf("parsed %d searches, indexed %d searches\n", len(searches), len(indexed))
}

func getSearchesFromLocalHtml() []string {
	html, err := os.ReadFile("data.html")
	Check(err)
	body := soup.HTMLParse(string(html))
	searches := body.FindAll("span", "ng-bind", "bidiText")
	uniques := map[string]bool{}
	for _, search := range searches {
		clean, ok := cleanSearch(search.Text())
		if ok {
			uniques[clean] = true
		}
	}
	keys := make([]string, len(uniques))
	i := 0
	for s, _ := range uniques {
		keys[i] = s
		i++
	}
	return keys
}

func cleanSearch(search string) (string, bool) {
	for _, edgeChar := range []rune{rune(search[0]), rune(search[len(search)-1])} {
		if !unicode.IsLetter(edgeChar) && !unicode.IsNumber(edgeChar) {
			return "", false
		}
	}
	var sb strings.Builder
	for _, char := range search {
		if !unicode.IsLetter(char) && !unicode.IsNumber(char) {
			sb.WriteRune(' ')
		} else {
			sb.WriteRune(char)
		}
	}
	return strings.ToLower(sb.String()), true
}
