package docs

// PageData represents the HTML page data
type PageData struct {
	Title       string
	Namespaces  []*NamespaceDoc
	TotalRoutes int
	CrumbNS     string
	Categories  []string
}

// RoutePageData represents a single route page data
type RoutePageData struct {
	// Title is required by the shared base template's <title> tag.
	Title            string
	Route            *RouteDoc
	Related          []*RouteDoc
	RouteEnvStatuses []CredStatus
}

// CredStatus groups the concrete cookies read from one env var.
type CredStatus struct {
	Key    string
	Fields []CredField
}

// CredField is one cookie and whether it is present in the env value.
type CredField struct {
	Name    string
	Present bool
}
