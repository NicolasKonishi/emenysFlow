package handlers

import "testing"

func TestBuildInventoryInternalCode(t *testing.T) {
	tests := []struct {
		prefix string
		name   string
		want   string
	}{
		{prefix: "CUB", name: "Cuba de Réchaud", want: "CUB-cuba-de-rechaud"},
		{prefix: " beb ", name: "Água  com gás 500 ml", want: "BEB-agua-com-gas-500-ml"},
		{prefix: "Louças", name: "Prato / sobremesa", want: "LOUCAS-prato-sobremesa"},
	}

	for _, test := range tests {
		if got := buildInventoryInternalCode(test.prefix, test.name); got != test.want {
			t.Errorf("buildInventoryInternalCode(%q, %q) = %q; want %q", test.prefix, test.name, got, test.want)
		}
	}
}
