package routes

import (
	"os"
	"strings"
	"testing"

	"github.com/xihale/rsshub-go/internal/testutil"
)

func TestEndOfLifeProductLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(EndOfLifeProductHandler, map[string]string{"product": "nodejs"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	if !strings.Contains(feed.Items[0].Title, "nodejs") || !strings.Contains(feed.Items[0].Title, "→") {
		t.Fatalf("unexpected first title: %q", feed.Items[0].Title)
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

func TestEndOfLifeInvalidProduct(t *testing.T) {
	if _, err := testutil.RunHandler(EndOfLifeProductHandler, map[string]string{"product": "../etc"}); err == nil {
		t.Fatal("expected error for invalid product slug")
	}
}

func TestEolFlexibleUnmarshal(t *testing.T) {
	var v eolFlexible
	if err := v.UnmarshalJSON([]byte(`"2029-04-30"`)); err != nil || v != "2029-04-30" {
		t.Fatalf("string: %q err=%v", v, err)
	}
	if err := v.UnmarshalJSON([]byte(`false`)); err != nil || v.String() != "No" {
		t.Fatalf("false: %q err=%v", v, err)
	}
	if err := v.UnmarshalJSON([]byte(`null`)); err != nil || v != "" {
		t.Fatalf("null: %q err=%v", v, err)
	}
}
