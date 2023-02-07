package main

import (
	"testing"
)

func TestShouldReturnEcogForKarnofsky100WithoutPercent(t *testing.T) {
	actual := karnofskyToEcog("100")
	expected := "0"
	if actual != expected {
		t.Logf("wrong ecog: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}

func TestShouldReturnEcogForKarnofsky100(t *testing.T) {
	actual := karnofskyToEcog("100%")
	expected := "0"
	if actual != expected {
		t.Logf("wrong ecog: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}

func TestShouldReturnEcogForKarnofsky90(t *testing.T) {
	actual := karnofskyToEcog("90%")
	expected := "0"
	if actual != expected {
		t.Logf("wrong ecog: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}

func TestShouldReturnEcogForKarnofsky80(t *testing.T) {
	actual := karnofskyToEcog("80%")
	expected := "1"
	if actual != expected {
		t.Logf("wrong ecog: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}

func TestShouldReturnEcogForKarnofsky70(t *testing.T) {
	actual := karnofskyToEcog("70%")
	expected := "1"
	if actual != expected {
		t.Logf("wrong ecog: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}

func TestShouldReturnEcogForKarnofsky60(t *testing.T) {
	actual := karnofskyToEcog("60%")
	expected := "2"
	if actual != expected {
		t.Logf("wrong ecog: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}

func TestShouldReturnEcogForKarnofsky50(t *testing.T) {
	actual := karnofskyToEcog("50%")
	expected := "2"
	if actual != expected {
		t.Logf("wrong ecog: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}

func TestShouldReturnEcogForKarnofsky40(t *testing.T) {
	actual := karnofskyToEcog("40%")
	expected := "3"
	if actual != expected {
		t.Logf("wrong ecog: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}

func TestShouldReturnEcogForKarnofsky30(t *testing.T) {
	actual := karnofskyToEcog("30%")
	expected := "3"
	if actual != expected {
		t.Logf("wrong ecog: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}

func TestShouldReturnEcogForKarnofsky20(t *testing.T) {
	actual := karnofskyToEcog("20%")
	expected := "4"
	if actual != expected {
		t.Logf("wrong ecog: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}

func TestShouldReturnEcogForKarnofsky10(t *testing.T) {
	actual := karnofskyToEcog("10%")
	expected := "4"
	if actual != expected {
		t.Logf("wrong ecog: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}

func TestShouldReturnEcogForKarnofsky0(t *testing.T) {
	actual := karnofskyToEcog("0%")
	expected := "5"
	if actual != expected {
		t.Logf("wrong ecog: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}
