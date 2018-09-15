package parser

import (
	"os"
	"reflect"
	"testing"

	"github.com/cube2222/usos-notifier/marks"
)

func TestGetClasses(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]*marks.Class
		wantErr bool
	}{
		{
			name: "",
			args: args{
				filename: "fixtures/classes/classes.html",
			},
			want: map[string]*marks.Class{
				"107749": {Name: "Analiza matematyczna inf. I"},
				"109476": {Name: "Geometria z algebrą liniową"},
				"109311": {Name: "Geometria z algebrą liniową"},
				"108034": {Name: "Podstawy matematyki"},
				"109067": {Name: "Wstęp do programowania (podejście funkcyjne)"},
				"115713": {Name: "Analiza matematyczna inf. II"},
				"116206": {Name: "Analiza matematyczna inf. II"},
				"117291": {Name: "Indywidualny projekt programistyczny"},
				"115987": {Name: "Matematyka dyskretna"},
				"119155": {Name: "Programowanie obiektowe"},
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

			got, err := GetClasses(f)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetClasses() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetClasses() = %v, want %v", got, tt.want)
			}
		})
	}
}
