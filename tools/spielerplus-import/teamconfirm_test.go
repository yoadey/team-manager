package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestConfirmTeam_ExpectedMatches(t *testing.T) {
	var out bytes.Buffer
	err := confirmTeam("TSC B-Team 25/26", "TSC B-Team 25/26", strings.NewReader(""), &out)
	if err != nil {
		t.Fatalf("confirmTeam() error = %v", err)
	}
}

func TestConfirmTeam_ExpectedMismatch(t *testing.T) {
	var out bytes.Buffer
	err := confirmTeam("TSC A-Team 25/26", "TSC B-Team 25/26", strings.NewReader(""), &out)
	if !errors.Is(err, ErrTeamNotConfirmed) {
		t.Fatalf("confirmTeam() error = %v, want ErrTeamNotConfirmed", err)
	}
}

func TestConfirmTeam_ExpectedIgnoresCaseAndSpace(t *testing.T) {
	var out bytes.Buffer
	err := confirmTeam("  tsc b-team 25/26  ", "TSC B-Team 25/26", strings.NewReader(""), &out)
	if err != nil {
		t.Fatalf("confirmTeam() error = %v", err)
	}
}

func TestConfirmTeam_InteractiveYes(t *testing.T) {
	for _, in := range []string{"y\n", "Y\n", "yes\n", "  yes  \n"} {
		var out bytes.Buffer
		if err := confirmTeam("TSC B-Team 25/26", "", strings.NewReader(in), &out); err != nil {
			t.Errorf("confirmTeam() with input %q error = %v, want nil", in, err)
		}
	}
}

func TestConfirmTeam_InteractiveNoOrEmpty(t *testing.T) {
	for _, in := range []string{"n\n", "no\n", "\n", ""} {
		var out bytes.Buffer
		err := confirmTeam("TSC B-Team 25/26", "", strings.NewReader(in), &out)
		if !errors.Is(err, ErrTeamNotConfirmed) {
			t.Errorf("confirmTeam() with input %q error = %v, want ErrTeamNotConfirmed", in, err)
		}
	}
}

func TestConfirmTeam_PrintsDetectedName(t *testing.T) {
	var out bytes.Buffer
	_ = confirmTeam("TSC B-Team 25/26", "", strings.NewReader("y\n"), &out)
	if !strings.Contains(out.String(), "TSC B-Team 25/26") {
		t.Errorf("output = %q, want it to mention the detected team name", out.String())
	}
}
