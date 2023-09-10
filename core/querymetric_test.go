package core_test

import (
	"strings"
	"testing"

	"github.com/sniffdogsniff/core"
	"github.com/sniffdogsniff/kademlia"
)

var testMetrics = []kademlia.KadId{
	kademlia.NewKadId("weapon"),
	kademlia.NewKadId("mass"),
	kademlia.NewKadId("destruction"),
	kademlia.NewKadId("wmd"),
}

func Test_ToQueryTokens(t *testing.T) {
	s := "A weapon of mass destruction (WMD) is a chemical,      biological, radiological or nuclear."

	split := core.ToQueryTokens(strings.ToLower(s))
	if len(split) > 8 {
		t.Fatal()
	}

	for i, w := range []string{"weapon", "mass", "destruction", "wmd", "chemical", "biological", "radiological", "nuclear"} {
		if split[i] != w {
			t.Fatalf("at eord %d: %s != %s", i, w, split[i])
		}
	}
}

func Test_Metric_NoDuplicates(t *testing.T) {
	s := "A weapon of mass destruction (WMD) is a chemical,      biological, radiological or nuclear."

	metrics := core.EvalQueryMetrics(s)
	if len(metrics) > 4 {
		t.Fatal()
	}

	for i, m := range testMetrics {
		if !metrics[i].Eq(m) {
			t.Fatalf("at eord %d: %s != %s", i, m, metrics[i])
		}
	}
}

func Test_Metric_WithDuplicates(t *testing.T) {
	s := "weapon A weapon of weapon of mass weapon is destruction weapon (WMD) mass destruction (WMD)"

	metrics := core.EvalQueryMetrics(s)
	if len(metrics) > 4 {
		t.Fatal()
	}

	for i, m := range testMetrics {
		if !metrics[i].Eq(m) {
			t.Fatalf("at eord %d: %s != %s", i, m, metrics[i])
		}
	}
}
