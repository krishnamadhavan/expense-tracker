package importers

import "testing"

func TestSampleCSV_Generic(t *testing.T) {
	name, body, ok := SampleCSV("generic_csv")
	if !ok || name == "" || len(body) < 20 {
		t.Fatalf("%v %q", ok, name)
	}
	if _, _, ok := SampleCSV("nope"); ok {
		t.Fatal("expected miss")
	}
}
