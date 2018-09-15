package parser

import (
	"io"
	"log"
	"regexp"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

type Class struct {
	Name string
}

var idRegexp = regexp.MustCompile("[0-9]+")

func GetClasses(r io.Reader) (map[string]*Class, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't parse html")
	}
	selection := doc.Find("[href^=\"https://usosweb.mimuw.edu.pl/kontroler.php?_action=dla_stud/studia/sprawdziany/pokaz\"]")

	classes := map[string]*Class{}
	for _, node := range selection.Nodes {
		id, class, err := getClass(node)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't parse class")
		}
		classes[id] = class
	}
	log.Printf("%+v", classes)

	return classes, nil
}

func getClass(node *html.Node) (id string, class *Class, err error) {
	defer func() {
		if recErr := recover(); recErr != nil {
			err = errors.Errorf("invalid class structure: %v", recErr)
		}
	}()

	id = idRegexp.FindString(getHref(node))
	if id == "" {
		return "", nil, errors.New("invalid ID")
	}

	return id, &Class{
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
