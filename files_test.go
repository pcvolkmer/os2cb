package main

import "testing"

func TestShouldReturnExpectedColumn(t *testing.T) {

	testsArgs := map[int]string{
		0:  "A",
		25: "Z",
		26: "AA",
		51: "AZ",
	}

	for key, value := range testsArgs {
		actual := getExcelColumn(key)
		if actual != value {
			t.Logf("wrong Column: Expected %s, got %s", value, actual)
			t.Fail()
		}
	}

}
