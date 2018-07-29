package main

import (
	"os"
	"reflect"
	"testing"

	"golang.org/x/net/html"
)

func Test_getSingleScore(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name      string
		args      args
		wantName  string
		wantScore *Score
		wantErr   bool
	}{
		// TODO: Add test cases.
		{
			name: "With a correct known score, I want a correct actual and max score as well as the name",
			args: args{
				filename: "single.html",
			},
			wantName: "zadanie 1",
			wantScore: &Score{
				known:  true,
				actual: 9.3,
				max:    10.2,
			},
			wantErr: false,
		},
		{
			name: "With a correct unknown score, I want a correct name, max score, and information that it's unknown",
			args: args{
				filename: "unknown.html",
			},
			wantName: "zadanie 1",
			wantScore: &Score{
				known: false,
				max:   10.2,
			},
			wantErr: false,
		},
		{
			name: "With a correct known score and a description present, I want a correct name, max score, and information that it's unknown",
			args: args{
				filename: "with_description.html",
			},
			wantName: "Liczba nieobecności nieuspr. na ćwiczeniach",
			wantScore: &Score{
				known: false,
				max:   40,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := loadTestNode(t, tt.args.filename)

			gotName, gotScore, err := getSingleScore(node)
			if (err != nil) != tt.wantErr {
				t.Errorf("getSingleScore() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotName != tt.wantName {
				t.Errorf("getSingleScore() gotName = %v, want %v", gotName, tt.wantName)
			}
			if !reflect.DeepEqual(gotScore, tt.wantScore) {
				t.Errorf("getSingleScore() gotScore = %v, want %v", gotScore, tt.wantScore)
			}
		})
	}
}

func Test_extractCategoryName(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "With a correct category name, I want to get it",
			args: args{
				filename: "category.html",
			},
			want:    "małe kolokwium",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := loadTestNode(t, tt.args.filename)

			// Going around some html.Parse addition (<html><head></head><body><table>...</table></body></html>)
			got, err := extractCategoryName(node.FirstChild.FirstChild.NextSibling.FirstChild)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractCategoryName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractCategoryName() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
