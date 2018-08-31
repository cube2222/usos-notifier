package parser

import (
	"os"
	"testing"

	"golang.org/x/net/html"
)

func loadTestNode(t *testing.T, filename string) *html.Node {
	f, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	node, err := html.Parse(f)
	if err != nil {
		t.Fatal(err)
	}

	return node
}
