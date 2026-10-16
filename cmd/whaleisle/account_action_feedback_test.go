package main

import (
	"errors"
	"testing"
)

func TestAccountLinkFailureFeedbackAndRetry(t *testing.T) {
	s := &mainState{}
	s.openAccountLink("https://example.test/top-up", true, func(string) error { return errors.New("browser unavailable") })
	if s.accountReturn.err == "" || s.accountReturn.external {
		t.Fatalf("failure not reflected: %+v", s.accountReturn)
	}
	s.openAccountLink("https://example.test/top-up", true, func(string) error { return nil })
	if s.accountReturn.err != "" || !s.accountReturn.external {
		t.Fatalf("successful retry not reflected: %+v", s.accountReturn)
	}
	s.openAccountLink("https://example.test/authorize", false, func(string) error { return errors.New("authorization open failed") })
	if s.accountReturn.err == "" {
		t.Fatal("authorization failure missing")
	}
	s.clearAccountReturn()
	if s.accountReturn.err != "" {
		t.Fatal("old Host error retained")
	}
}
