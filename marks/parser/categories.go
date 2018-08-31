package parser

import (
	"io"
	"log"
	"regexp"

	"github.com/PuerkitoBio/goquery"
	"github.com/cube2222/usos-notifier/marks"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

var idRegexp = regexp.MustCompile("[0-9]+")

func GetCategories(r io.Reader) (map[string]*marks.Category, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't parse html")
	}
	selection := doc.Find("[href^=\"https://usosweb.mimuw.edu.pl/kontroler.php?_action=dla_stud/studia/sprawdziany/pokaz\"]")

	categories := map[string]*marks.Category{}
	for _, node := range selection.Nodes {
		id, category, err := getCategory(node)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't parse category")
		}
		categories[id] = category
	}
	log.Printf("%+v", categories)

	return categories, nil
}

func getCategory(node *html.Node) (id string, category *marks.Category, err error) {
	defer func() {
		if recErr := recover(); recErr != nil {
			err = errors.Errorf("invalid category structure: %v", recErr)
		}
	}()

	id = idRegexp.FindString(getHref(node))
	if id == "" {
		return "", nil, errors.New("invalid ID")
	}

	return id, &marks.Category{
		Name: node.FirstChild.Data,
	}, nil
}

func getHref(node *html.Node) string {
	for _, attr := range node.Attr {
		if attr.Key == "href" {
			return attr.Val
		}
	}

	return ""
}
