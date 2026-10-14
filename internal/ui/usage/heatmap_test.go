package usage

import "testing"

func TestHeatCellIntensityOrder(t *testing.T) {
	cells := []HeatCell{{Label: "1/1", Value: 0}, {Label: "1/2", Value: 3}}
	if cells[0].Value >= cells[1].Value {
		t.Fatalf("expected empty day weaker than active day")
	}
}
