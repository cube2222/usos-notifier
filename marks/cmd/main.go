package main

import (
	"log"
	"os"

	"github.com/PuerkitoBio/goquery"
	"github.com/cube2222/usos-notifier/marks/parser"
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

	scores := make(map[string]*parser.Score)
	err = parser.GetScores(scores, "", node)
	if err != nil {
		log.Fatal(err)
	}

	for k, v := range scores {
		log.Printf("%s: %v", k, v.Actual)
	}

	/*err = ioutil.WriteFile("test2.html", []byte(out), os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}*/
}
