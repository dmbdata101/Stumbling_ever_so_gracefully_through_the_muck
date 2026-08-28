package picker

import (
	"strings"
	"testing"

	"github.com/dmbdata101/lab/internal/proto"
)

var devices = []proto.Choice{
	{ID: "r1", Label: "r1  ospf  172.20.20.11"},
	{ID: "r2", Label: "r2  ospf  172.20.20.12"},
	{ID: "sw1", Label: "sw1  vlan  172.20.20.11"},
}

func TestMatchPrefixUnique(t *testing.T) {
	got, err := matchPrefix("sw", devices)
	if err != nil {
		t.Fatalf("matchPrefix: %v", err)
	}
	if got != "sw1" {
		t.Errorf("got %q, want sw1", got)
	}
}

func TestMatchPrefixIsCaseInsensitive(t *testing.T) {
	if got, err := matchPrefix("SW1", devices); err != nil || got != "sw1" {
		t.Errorf("got %q, %v", got, err)
	}
}

// Guessing between two matches is worse than asking again.
func TestMatchPrefixAmbiguous(t *testing.T) {
	_, err := matchPrefix("ospf", devices)
	if err == nil {
		t.Fatal("expected an ambiguity error")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("error = %v", err)
	}
}

func TestMatchPrefixNoHit(t *testing.T) {
	if _, err := matchPrefix("zzz", devices); err == nil {
		t.Fatal("expected an error for no match")
	}
}

// One option is not a choice worth interrupting for.
func TestChooseSingleSkipsPrompt(t *testing.T) {
	got, err := Choose("pick", devices[:1])
	if err != nil {
		t.Fatalf("Choose: %v", err)
	}
	if got != "r1" {
		t.Errorf("got %q, want r1", got)
	}
}

func TestChooseEmpty(t *testing.T) {
	if _, err := Choose("pick", nil); err == nil {
		t.Fatal("expected an error with nothing to choose from")
	}
}
