package parser

import (
	"os"
	"reflect"
	"testing"

	"github.com/cube2222/usos-notifier/marks"
	"golang.org/x/net/html"
)

func TestGetScores(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]*marks.Score
		wantErr bool
	}{
		{
			name: "",
			args: args{
				filename: "fixtures/full/test.html",
			},
			want: map[string]*marks.Score{
				"Duże kolokwium/Zadanie 1":                                                {Unknown: false, Hidden: false, Actual: 3, Max: 17.5},
				"kartkówki (max 30)/kartkówka 2":                                          {Unknown: false, Hidden: false, Actual: 2.5, Max: 5},
				"Egzamin/Termin I/pisemny/Zadanie I.3":                                    {Unknown: false, Hidden: false, Actual: 7.5, Max: 15},
				"Egzamin/Termin II - poprawkowy/pisemny/Zadanie I.3":                      {Unknown: false, Hidden: true, Actual: 0, Max: 15},
				"Egzamin/Termin II - poprawkowy/pisemny/Zadanie II.4":                     {Unknown: false, Hidden: true, Actual: 0, Max: 15},
				"Duże kolokwium/Zadanie 2":                                                {Unknown: false, Hidden: false, Actual: 11.5, Max: 17.5},
				"Egzamin/Termin II - poprawkowy/pisemny/Zadanie I.1":                      {Unknown: false, Hidden: true, Actual: 0, Max: 15},
				"Duże kolokwium/Zadanie 3":                                                {Unknown: false, Hidden: false, Actual: 0, Max: 17.5},
				"kartkówki (max 30)/kartkówka 3":                                          {Unknown: false, Hidden: false, Actual: 5, Max: 5},
				"od prow. ćwiczenia/suma punktów od prow. ćw.":                            {Unknown: false, Hidden: false, Actual: 70, Max: 70},
				"Egzamin/Termin I/pisemny/Zadanie I.2":                                    {Unknown: false, Hidden: false, Actual: 14, Max: 15},
				"Egzamin/Termin I/pisemny/Zadanie II.2":                                   {Unknown: false, Hidden: false, Actual: 15, Max: 15},
				"Egzamin/Termin II - poprawkowy/pisemny/Zadanie I.4":                      {Unknown: false, Hidden: true, Actual: 0, Max: 15},
				"małe kolokwium/zadanie 2":                                                {Unknown: false, Hidden: false, Actual: 5, Max: 10},
				"Egzamin/Termin II - poprawkowy/pisemny/Zadanie II.1":                     {Unknown: false, Hidden: true, Actual: 0, Max: 15},
				"Duże kolokwium/Zadanie 4":                                                {Unknown: false, Hidden: false, Actual: 12, Max: 17.5},
				"kartkówki (max 30)/kartkówka 1":                                          {Unknown: false, Hidden: false, Actual: 5, Max: 5},
				"Egzamin/Termin II - poprawkowy/pisemny/Zadanie II.2":                     {Unknown: false, Hidden: true, Actual: 0, Max: 15},
				"Egzamin/Termin II - poprawkowy/pisemny/Zadanie II.3":                     {Unknown: false, Hidden: true, Actual: 0, Max: 15},
				"kartkówki (max 30)/kartkówka 5":                                          {Unknown: false, Hidden: false, Actual: 2.5, Max: 5},
				"od prow. ćwiczenia/\"dopytka wariant 1\" - tylko gdy pomyślna wpisz 100": {Unknown: true, Hidden: false, Actual: 0, Max: 100},
				"Egzamin/Termin I/pisemny/Zadanie II.3":                                   {Unknown: false, Hidden: false, Actual: 14, Max: 15},
				"Egzamin/Termin I/pisemny/Zadanie I.1":                                    {Unknown: false, Hidden: false, Actual: 12, Max: 15},
				"Egzamin/Termin I/pisemny/Zadanie I.4":                                    {Unknown: false, Hidden: false, Actual: 9, Max: 15},
				"małe kolokwium/zadanie 1":                                                {Unknown: false, Hidden: false, Actual: 10, Max: 10},
				"małe kolokwium/zadanie 3":                                                {Unknown: false, Hidden: false, Actual: 10, Max: 10},
				"kartkówki (max 30)/kartkówka 4":                                          {Unknown: false, Hidden: false, Actual: 5, Max: 5},
				"kartkówki (max 30)/kartkówka 6":                                          {Unknown: false, Hidden: false, Actual: 0, Max: 5},
				"od prow. ćwiczenia/Liczba nieobecności nieuspr. na ćwiczeniach":          {Unknown: true, Hidden: false, Actual: 0, Max: 40},
				"od prow. ćwiczenia/Czy należy się \"dopytka wariant 1\"? (1=TAK!)":       {Unknown: true, Hidden: false, Actual: 0, Max: 1},
				"Egzamin/Termin I/pisemny/Zadanie II.4":                                   {Unknown: false, Hidden: false, Actual: 11, Max: 15},
				"Egzamin/Termin II - poprawkowy/pisemny/Zadanie I.2":                      {Unknown: false, Hidden: true, Actual: 0, Max: 15},
				"Egzamin/Termin I/pisemny/Zadanie II.1":                                   {Unknown: false, Hidden: false, Actual: 15, Max: 15},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(tt.args.filename)
			if err != nil {
				t.Fatal(err)
			}

			got, err := GetScores(f)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetScores() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetScores() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getSingleScore(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name      string
		args      args
		wantName  string
		wantScore *marks.Score
		wantErr   bool
	}{
		// TODO: Add test cases.
		{
			name: "With a correct unknown score, I want a correct actual and max score as well as the name",
			args: args{
				filename: "fixtures/single/single.html",
			},
			wantName: "zadanie 1",
			wantScore: &marks.Score{
				Actual: 9.3,
				Max:    10.2,
			},
			wantErr: false,
		},
		{
			name: "With a correct unknown score, I want a correct name, max score, and information that it's unknown",
			args: args{
				filename: "fixtures/single/unknown.html",
			},
			wantName: "zadanie 1",
			wantScore: &marks.Score{
				Unknown: true,
				Max:     10.2,
			},
			wantErr: false,
		},
		{
			name: "With a correct unknown score and a description present, I want a correct actual and max score as well as the name",
			args: args{
				filename: "fixtures/single/with_description.html",
			},
			wantName: "Liczba nieobecności nieuspr. na ćwiczeniach",
			wantScore: &marks.Score{
				Unknown: true,
				Max:     40,
			},
			wantErr: false,
		},
		{
			name: "With a hidden score, I want a correct name, max score, and information that it's hidden.",
			args: args{
				filename: "fixtures/single/hidden.html",
			},
			wantName: "Zadanie I.1",
			wantScore: &marks.Score{
				Hidden: true,
				Max:    15.0,
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
				filename: "fixtures/category/category.html",
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
