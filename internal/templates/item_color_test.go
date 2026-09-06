package templates

import "testing"

func TestItemColor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		notes string
		want  string
	}{
		{"", ""},
		{"Levar caixa extra", ""},
		{"Cor: Dourada", "Dourada"},
		{"cor: azul royal", "azul royal"},
		{"Montar na entrada\nCor: Branca", "Branca"},
	}

	for _, tc := range cases {
		if got := itemColor(tc.notes); got != tc.want {
			t.Fatalf("itemColor(%q)=%q want %q", tc.notes, got, tc.want)
		}
	}
}
