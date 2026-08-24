package routes

import (
	"os"
	"testing"

	"github.com/xihale/syndi/internal/testutil"
)

func TestToutiaoUserLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(ToutiaoUserHandler, map[string]string{
		"token": "MS4wLjABAAAA_Q07NxeCa4hDPFoRcdphaZOk2X6C8BApfpTPTMLJswI",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected items")
	}
	if feed.Title == "" || feed.Link == "" {
		t.Fatal("expected normalized feed title/link")
	}
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}

// TestSM3Vector checks the SM3 implementation against the standard test vector.
func TestSM3Vector(t *testing.T) {
	// SM3("abc") per GB/T 32905-2016
	got := sm3Sum([]byte("abc"))
	want := [32]byte{
		0x66, 0xc7, 0xf0, 0xf4, 0x62, 0xee, 0xed, 0xd9, 0xd1, 0xf2, 0xd4, 0x6b, 0xdc, 0x10, 0xe4, 0xe2,
		0x41, 0x67, 0xc4, 0x87, 0x5c, 0xf2, 0xf7, 0xa2, 0x29, 0x7d, 0xa0, 0x2b, 0x8f, 0x4b, 0xa8, 0xe0,
	}
	if got != want {
		t.Fatalf("sm3 mismatch: got %x want %x", got, want)
	}
}
