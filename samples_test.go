package main

import "testing"

func TestSanitizeSampleId(t *testing.T) {
	actual := sanitizeSampleId("H/2024/1234")
	expected := "H1234-24"
	if actual != expected {
		t.Logf("wrong value: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}

func TestKeepOtherFormatSampleId(t *testing.T) {
	actual := sanitizeSampleId("H-2024-1234")
	expected := "H-2024-1234"
	if actual != expected {
		t.Logf("wrong value: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}
