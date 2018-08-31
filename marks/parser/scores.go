package parser

import (
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/cube2222/usos-notifier/marks"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

// GetScores takes the whole html website with scores, returning the scores.
func GetScores(r io.Reader) (map[string]*marks.Score, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		log.Fatal(err) //TODO: Get rid of this fatal
	}

	nodes := doc.Find("[id^='childrenof']").Nodes
	if len(nodes) < 1 {
		log.Println(len(nodes))
		return nil, errors.Wrap(err, "couldn't get top-level childrenof*")
	}
	node := nodes[0]

	scores := make(map[string]*marks.Score)
	err = getScores(scores, "", node)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get scores")
	}

	return scores, nil
}

func getScores(scores map[string]*marks.Score, prefix string, node *html.Node) error {
	foundSubTrees := false

	children := getElementNodeChildren(node)

	if len(children) == 0 {
		return errors.New("No children.")
	}

	for i, cur := range children {
		if cur.Data != "div" {
			continue
		}
		if strings.HasPrefix(getId(cur), "childrenof") {
			foundSubTrees = true
			if i == 0 {
				return errors.New("unexpected category tree without category name")
			}
			category, err := extractCategoryName(children[i-1])
			if err != nil {
				return errors.Wrap(err, "couldn't get category name")
			}
			err = getScores(scores, fmt.Sprintf("%s%s/", prefix, category), cur)
			if err != nil {
				return errors.Wrap(err, "couldn't get scores")
			}
		}
	}

	if !foundSubTrees {
		for _, cur := range children {
			if cur.Data != "table" {
				continue
			}

			name, score, err := getSingleScore(cur)
			if err != nil {
				return errors.Wrapf(err, "couldn't get single score, prefix \"%s\"", prefix)
			}

			scores[fmt.Sprintf("%s%s", prefix, name)] = score
		}
	}

	return nil
}

func getElementNodeChildren(node *html.Node) []*html.Node {
	children := make([]*html.Node, 0)
	for cur := node.FirstChild; cur != nil; cur = cur.NextSibling {
		if cur.Type == html.ElementNode {
			children = append(children, cur)
		}
	}
	return children
}

func getId(node *html.Node) string {
	for _, attr := range node.Attr {
		if attr.Key == "id" {
			return attr.Val
		}
	}

	return ""
}

func extractCategoryName(node *html.Node) (name string, err error) {
	defer func() {
		if recErr := recover(); recErr != nil {
			err = errors.Errorf("invalid score structure: %v", recErr)
		}
	}()

	doc := goquery.NewDocumentFromNode(node).Find(".strong")
	if len(doc.Nodes) != 1 {
		return "", errors.Errorf("unexpected node count: %v expected: %v", len(doc.Nodes), 1)
	}

	// we're in the td node, get the first child, the text
	name = strings.TrimSpace(doc.Nodes[0].FirstChild.Data)

	return name, nil
}

var maxRegexp = regexp.MustCompile("[0-9]+(\\.[0-9]+)?")

func getSingleScore(node *html.Node) (name string, score *marks.Score, err error) {
	defer func() {
		if recErr := recover(); recErr != nil {
			err = errors.Errorf("invalid score structure: %v", recErr)
		}
	}()

	nodes := goquery.NewDocumentFromNode(node).Find("tr").Nodes
	if len(nodes) < 1 {
		return "", nil, errors.New("malformed html, couldn't find any <tr> tag")
	}
	children := getElementNodeChildren(nodes[0])

	name = strings.TrimSpace(children[1].FirstChild.Data) // Second td, the text
	max, err := strconv.ParseFloat(
		maxRegexp.FindString(
			children[1].FirstChild.NextSibling.FirstChild.Data,
		),
		64,
	)                                                     // Second td, subspan text
	if err != nil {
		return "", nil, errors.Wrap(err, "invalid max score")
	}

	scoreString := children[2].FirstChild.NextSibling.FirstChild.Data
	if scoreString == "brak wyniku" {
		return name, &marks.Score{
			Unknown: true,
			Max:     max,
		}, nil
	}
	if scoreString == "wynik jest ukryty" {
		return name, &marks.Score{
			Hidden: true,
			Max:    max,
		}, nil
	}

	actual, err := strconv.ParseFloat(
		scoreString,
		64,
	) // Third td, sub-b text
	if err != nil {
		return "", nil, errors.Wrap(err, "invalid actual score")
	}

	return name, &marks.Score{
		Actual: actual,
		Max:    max,
	}, nil
}
