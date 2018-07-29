package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

func main() {
	/*req, err := http.NewRequest(http.MethodGet, "https://usosweb.mimuw.edu.pl/kontroler.php?_action=dla_stud/studia/sprawdziany/pokaz&wez_id=115713", nil)
	if err != nil {
		log.Fatal(err)
	}

	req.AddCookie(
		&http.Cookie{
			Name:       "PHPSESSID",
			Value:      "",
			Path:       "/",
			Domain:     "usosweb.mimuw.edu.pl",
			Expires:    time.Now().Add(time.Minute * 15),
		},
	)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()*/

	f, err := os.Open("test.html")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		log.Fatal(err)
	}

	nodes := doc.Find("[id='childrenof115713']").Nodes
	if len(nodes) != 1 {
		log.Fatal("Malformed page.")
	}
	node := nodes[0]
	log.Printf("%+v", node)

	scores := make(map[string]*Score)
	err = getScores(scores, "", node)
	if err != nil {
		for k, v := range scores {
			log.Printf("%s: %v", k, v.actual)
		}
		log.Fatal(err)
	}

	/*err = ioutil.WriteFile("test2.html", []byte(out), os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}*/
}

func getScores(scores map[string]*Score, prefix string, node *html.Node) error {
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

	// we're in the td node, get the firs child, the text
	name = strings.TrimSpace(doc.Nodes[0].FirstChild.Data)

	return name, nil
}

var maxRegexp = regexp.MustCompile("[0-9]+(\\.[0-9]+)?")

type Score struct {
	known       bool
	actual, max float64
}

func getSingleScore(node *html.Node) (name string, score *Score, err error) {
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
	) // Second td, subspan text
	if err != nil {
		return "", nil, errors.Wrap(err, "invalid max score")
	}

	scoreString := children[2].FirstChild.NextSibling.FirstChild.Data
	if scoreString == "brak wyniku" {
		return name, &Score{
			known: false,
			max:   max,
		}, nil
	}

	actual, err := strconv.ParseFloat(
		scoreString,
		64,
	) // Third td, sub-b text
	if err != nil {
		return "", nil, errors.Wrap(err, "invalid actual score")
	}

	return name, &Score{
		known:  true,
		actual: actual,
		max:    max,
	}, nil
}
