package toolkit

import "testing"

func TestTools_RandomString(t *testing.T) {
	var testingTools Tools

	s := testingTools.RandomString(10)

	if len(s) != 10 {
		t.Error("Wrong Length of random string")
	}
}
