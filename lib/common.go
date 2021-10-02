package lib

const INDEX = "searches"
const TYPE = "search"

type SearchDoc struct {
	Search string `json:"search"`
}

func Check(e error) {
	if e != nil {
		panic(e)
	}
}
