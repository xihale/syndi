package routes

import "testing"

func TestParseRedditSort(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "", want: "hot"},
		{in: "new", want: "new"},
		{in: "TOP", want: "top"},
		{in: "rising", want: "rising"},
		{in: "unknown", want: "hot"},
	}

	for _, tc := range tests {
		got := parseRedditSort(tc.in)
		if got != tc.want {
			t.Fatalf("parseRedditSort(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseRedditLimit(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{in: "", want: 25},
		{in: "10", want: 10},
		{in: "200", want: 100},
		{in: "-1", want: 25},
		{in: "abc", want: 25},
	}

	for _, tc := range tests {
		got := parseRedditLimit(tc.in)
		if got != tc.want {
			t.Fatalf("parseRedditLimit(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestBuildRedditListingURL(t *testing.T) {
	got := buildRedditListingURL("golang", "top", 15, "week")
	want := "https://www.reddit.com/r/golang/top/.rss?t=week"
	if got != want {
		t.Fatalf("buildRedditListingURL() = %q, want %q", got, want)
	}
}

func TestResolveRedditPostLink(t *testing.T) {
	tests := []struct {
		name      string
		postURL   string
		permalink string
		want      string
	}{
		{
			name:      "uses absolute url",
			postURL:   "https://example.com/a",
			permalink: "/r/golang/comments/abc",
			want:      "https://example.com/a",
		},
		{
			name:      "expands relative url",
			postURL:   "/r/golang/comments/abc",
			permalink: "/r/golang/comments/abc",
			want:      "https://www.reddit.com/r/golang/comments/abc",
		},
		{
			name:      "falls back to permalink",
			postURL:   "",
			permalink: "/r/golang/comments/abc",
			want:      "https://www.reddit.com/r/golang/comments/abc",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveRedditPostLink(tc.postURL, tc.permalink)
			if got != tc.want {
				t.Fatalf("resolveRedditPostLink(%q, %q) = %q, want %q", tc.postURL, tc.permalink, got, tc.want)
			}
		})
	}
}

func TestBuildRedditDescription(t *testing.T) {
	post := RedditPost{
		Selftext:     "plain",
		SelftextHTML: "&lt;p&gt;hello&lt;/p&gt;",
	}
	got := buildRedditDescription(post)
	if got != "<p>hello</p>" {
		t.Fatalf("buildRedditDescription() = %q, want %q", got, "<p>hello</p>")
	}
}
