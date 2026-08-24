package routes

import "time"

// GitLabProject is the "simple=true" project payload from the projects API.
type GitLabProject struct {
	Name           string    `json:"name"`
	PathWithSpace  string    `json:"path_with_namespace"`
	WebURL         string    `json:"web_url"`
	Description    string    `json:"description"`
	LastActivityAt time.Time `json:"last_activity_at"`
	StarCount      int       `json:"star_count"`
}
