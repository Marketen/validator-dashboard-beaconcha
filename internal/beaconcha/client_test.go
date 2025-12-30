package beaconcha

import "testing"

func TestParseBigToHuman(t *testing.T) {
	g, e := ParseBigToHuman("1000000000000000000") // 1 ETH in wei
	if g == "0" || e == "0" {
		t.Fatalf("unexpected parse result: g=%s e=%s", g, e)
	}
}
