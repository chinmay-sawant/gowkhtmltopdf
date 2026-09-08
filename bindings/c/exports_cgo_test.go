//go:build cgo

package main

import (
	"strings"
	"testing"
)

func TestConvertAllowListRejectsOversized(t *testing.T) {
	t.Parallel()

	got, ok := probeAllowListTooLong()
	if ok || got != nil {
		t.Fatalf("convertAllowList(oversized) = (%v, %v), want (nil, false)", got, ok)
	}
}

func TestConvertAllowListCopies(t *testing.T) {
	t.Parallel()

	got, ok := probeAllowListCopies([]string{"alpha", "", "gamma"}, map[int]bool{1: true})
	if !ok {
		t.Fatal("convertAllowList must accept a capped length")
	}
	if len(got) != 2 || got[0] != "alpha" || got[1] != "gamma" {
		t.Fatalf("convertAllowList = %v, want [alpha gamma]", got)
	}

	got, ok = probeAllowListCopies([]string{"only"}, nil)
	if !ok || len(got) != 1 || got[0] != "only" {
		t.Fatalf("convertAllowList(single) = (%v, %v), want ([only], true)", got, ok)
	}
}

func TestFinishResultNilOut(t *testing.T) {
	t.Parallel()

	for name, flags := range map[string][3]bool{
		"nil cOutData": {true, false, false},
		"nil cOutLen":  {false, true, false},
		"nil cErr":     {false, false, true},
	} {
		t.Run(name, func(t *testing.T) {
			status, _, _ := probeFinishResult(statusOK, []byte("x"), "", flags[0], flags[1], flags[2])
			if status != statusInvalidArg {
				t.Fatalf("finishResult = %d, want %d", status, statusInvalidArg)
			}
		})
	}
}

func TestFinishResultSuccessAndFailure(t *testing.T) {
	t.Parallel()

	status, outLen, msg := probeFinishResult(statusOK, []byte("payload"), "", false, false, false)
	if status != statusOK || outLen != 7 || msg != "" {
		t.Fatalf("finishResult(success) = (%d, %d, %q), want (0, 7, \"\")", status, outLen, msg)
	}

	status, outLen, msg = probeFinishResult(statusRenderError, nil, "", false, false, false)
	if status != statusRenderError || outLen != 0 || msg == "" {
		t.Fatalf("finishResult(failure) = (%d, %d, %q)", status, outLen, msg)
	}

	status, _, msg = probeFinishResult(statusOK, nil, "", false, false, false)
	if status != statusRenderError || msg == "" {
		t.Fatalf("finishResult(empty) = (%d, %q), want statusRenderError", status, msg)
	}
}

func TestAllowListMessage(t *testing.T) {
	t.Parallel()

	msg := allowListMessage(maxAllowEntries + 1)
	if !strings.Contains(msg, "1025") || !strings.Contains(msg, "1024") {
		t.Fatalf("allowListMessage = %q, want it to name got and limit", msg)
	}
}
