package lib

const INDEX = "searches"
const SEPARATOR_STR = " "
const SEPARATOR_RUNE = ' '

type SearchDoc struct {
	Search string `json:"search"`
}

func Check(e error) {
	if e != nil {
		panic(e)
	}
}
