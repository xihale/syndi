#!/usr/bin/env bash
set -euo pipefail

usage() {
	cat <<'EOF'
Usage:
  ./scripts/new-route.sh <namespace> <file> <route_path> <route_name> <example>

Example:
  ./scripts/new-route.sh github stars stars/:owner "GitHub Stars" "github/stars/octocat"

Notes:
  - <namespace> maps to routes/<namespace>/
  - <file> is the route file name without .go (supports letters, numbers, _ and -)
  - <route_path> is relative to the namespace (omit the leading /namespace/)
  - The script updates routes/<namespace>/routes.go for automatic registration
EOF
}

if [[ $# -ne 5 ]]; then
	usage
	exit 1
fi

namespace="$1"
file_name="$2"
route_path="$3"
route_name="$4"
example="$5"

if [[ "$route_path" =~ ^/ ]]; then
	normalized="${route_path#/}"
	if [[ "$normalized" == "$namespace" ]]; then
		route_path=""
	elif [[ "$normalized" == "$namespace/"* ]]; then
		route_path="${normalized#${namespace}/}"
	else
		route_path="$normalized"
	fi
fi

if [[ ! "$namespace" =~ ^[a-z0-9_]+$ ]]; then
	echo "Error: namespace must match ^[a-z0-9_]+$: $namespace" >&2
	exit 1
fi

if [[ ! "$file_name" =~ ^[a-zA-Z0-9_-]+$ ]]; then
	echo "Error: file must match ^[a-zA-Z0-9_-]+$: $file_name" >&2
	exit 1
fi

to_pascal_case() {
	local input="$1"
	echo "$input" | sed -E 's/[^a-zA-Z0-9]+/ /g' | awk '
	{
		out = ""
		for (i = 1; i <= NF; i++) {
			word = $i
			first = toupper(substr(word, 1, 1))
			rest = substr(word, 2)
			out = out first rest
		}
		printf "%s", out
	}'
}

package_name="routes"
if [[ "$namespace" == "test" ]]; then
	package_name="test"
fi

safe_file="${file_name//-/_}"
route_dir="routes/$namespace"
route_file="$route_dir/$safe_file.go"
test_file="$route_dir/${safe_file}_test.go"
routes_file="$route_dir/routes.go"

if [[ -f "$route_file" ]]; then
	echo "Error: route file already exists: $route_file" >&2
	exit 1
fi

mkdir -p "$route_dir"

handler_base="$(to_pascal_case "${namespace}_${safe_file}")"
handler_name="${handler_base}Handler"
route_var_name="$(echo "${handler_base}Route" | awk '{print tolower(substr($0,1,1)) substr($0,2)}')"

cat >"$route_file" <<EOF
package $package_name

import (
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var $route_var_name = routeutils.RouteSpec{
	Path:        "$route_path",
	Name:        "$route_name",
	Example:     "$example",
	Maintainers: []string{"yourname"},
	Description: "TODO: add route description",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{},
	CacheTTL:    30 * time.Minute,
	Handler:     $handler_name,
}

// $handler_name handles $route_path
func $handler_name(c *ctxpkg.Context) (*models.Feed, error) {
	feed := routeutils.NewFeed(
		"$route_name",
		c.BaseURL()+c.Req.URL.Path,
		"TODO: fill feed description",
	)
	return feed, nil
}
EOF

cat >"$test_file" <<EOF
package $package_name

import "testing"

func Test${handler_base}RouteSpec(t *testing.T) {
	if $route_var_name.Path != "$route_path" {
		t.Fatalf("unexpected path: %s", $route_var_name.Path)
	}
	if $route_var_name.Handler == nil {
		t.Fatal("handler should not be nil")
	}
}
EOF

if [[ ! -f "$routes_file" ]]; then
	cat >"$routes_file" <<EOF
package $package_name

import "github.com/xihale/syndi/internal/routeutils"

// Routes lists all $namespace route specs in this package.
var Routes = []routeutils.RouteSpec{
	$route_var_name,
}
EOF
else
	if ! grep -Fq "$route_var_name" "$routes_file"; then
		tmp_file="$(mktemp)"
		if ! awk -v line="\t$route_var_name," '
			BEGIN { in_routes = 0; inserted = 0 }
			/^var Routes =/ { in_routes = 1 }
			in_routes && !inserted && /^[[:space:]]*\}/ {
				print line
				inserted = 1
			}
			{ print }
			END {
				if (!inserted) {
					exit 2
				}
			}
		' "$routes_file" >"$tmp_file"; then
			rm -f "$tmp_file"
			echo "Error: failed to update $routes_file. Please add $route_var_name to Routes manually." >&2
			exit 1
		fi
		mv "$tmp_file" "$routes_file"
	fi
fi

gofmt -w "$route_file" "$test_file" "$routes_file"
go run scripts/generate-routes.go >/dev/null

echo "Created:"
echo "  - $route_file"
echo "  - $test_file"
if [[ -f "$routes_file" ]]; then
	echo "  - $routes_file (updated)"
fi
echo
echo "Next:"
echo "  1) Implement $handler_name in $route_file"
echo "  2) Update categories/description/params/TTL metadata in $route_file"
echo "  3) Run: go test ./..."
