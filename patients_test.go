package main

import (
	"testing"
)

func TestShouldReturnEcogForKarnovsky100WithoutPercent(t *testing.T) {
	actual := karnovskyToEcog("100")
	expected := "0"
	if actual != expected {
		t.Logf("wrong ecog: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}

func TestShouldReturnEcogForKarnovsky100(t *testing.T) {
	actual := karnovskyToEcog("100%")
	expected := "0"
	if actual != expected {
		t.Logf("wrong ecog: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}

func TestShouldReturnEcogForKarnovsky90(t *testing.T) {
	actual := karnovskyToEcog("90%")
	expected := "0"
	if actual != expected {
		t.Logf("wrong ecog: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}

func TestShouldReturnEcogForKarnovsky80(t *testing.T) {
	actual := karnovskyToEcog("80%")
	expected := "1"
	if actual != expected {
		t.Logf("wrong ecog: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}

func TestShouldReturnEcogForKarnovsky70(t *testing.T) {
	actual := karnovskyToEcog("70%")
	expected := "1"
	if actual != expected {
		t.Logf("wrong ecog: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}

func TestShouldReturnEcogForKarnovsky60(t *testing.T) {
	actual := karnovskyToEcog("60%")
	expected := "2"
	if actual != expected {
		t.Logf("wrong ecog: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}

func TestShouldReturnEcogForKarnovsky50(t *testing.T) {
	actual := karnovskyToEcog("50%")
	expected := "2"
	if actual != expected {
		t.Logf("wrong ecog: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}

func TestShouldReturnEcogForKarnovsky40(t *testing.T) {
	actual := karnovskyToEcog("40%")
	expected := "3"
	if actual != expected {
		t.Logf("wrong ecog: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}

func TestShouldReturnEcogForKarnovsky30(t *testing.T) {
	actual := karnovskyToEcog("30%")
	expected := "3"
	if actual != expected {
		t.Logf("wrong ecog: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}

func TestShouldReturnEcogForKarnovsky20(t *testing.T) {
	actual := karnovskyToEcog("20%")
	expected := "4"
	if actual != expected {
		t.Logf("wrong ecog: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}

func TestShouldReturnEcogForKarnovsky10(t *testing.T) {
	actual := karnovskyToEcog("10%")
	expected := "4"
	if actual != expected {
		t.Logf("wrong ecog: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}

func TestShouldReturnEcogForKarnovsky0(t *testing.T) {
	actual := karnovskyToEcog("0%")
	expected := "5"
	if actual != expected {
		t.Logf("wrong ecog: Expected %s, got %s", expected, actual)
		t.Fail()
	}
}
