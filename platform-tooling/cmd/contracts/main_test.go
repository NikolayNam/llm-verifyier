package main

import "testing"

func TestCheckCertificates(t *testing.T) {
	if err := checkCertificates(); err != nil {
		t.Fatalf("checkCertificates() error = %v", err)
	}
}

func TestCheckNaturalDeduction(t *testing.T) {
	if err := checkNaturalDeduction(); err != nil {
		t.Fatalf("checkNaturalDeduction() error = %v", err)
	}
}
