// Command genrbac parses backend/openapi/openapi.yaml and emits
// internal/middleware/rbac_table.gen.go: a table mapping every team-scoped
// operation (a path containing "{teamId}") to its RBAC module and
// self-service classification, read from the operation's x-rbac-module /
// x-rbac-self-service extensions.
//
// This is the single source of truth for route-to-module mapping described
// in openspec/changes/generate-rbac-from-spec: every team-scoped operation
// MUST carry x-rbac-module, or genrbac fails the build (make generate)
// rather than silently defaulting -- this is what keeps the fail-open GET
// bug the change fixes from being able to recur for a newly added route.
package main

import (
	"errors"
	"fmt"
	"go/format"
	"os"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

var (
	errNoPathsMapping     = errors.New("openapi.yaml: no top-level 'paths' mapping found")
	errNoOperationID      = errors.New("operation has no operationId")
	errUnrecognizedModule = errors.New("x-rbac-module is not a recognized module")
	errMissingModule      = errors.New("team-scoped operations are missing x-rbac-module (every team-scoped operation must carry one -- use \"public\" for membership-only routes)")
)

var httpMethods = []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"}

// validModules are the RBAC modules routeEntry.Module may take, plus the
// "public" sentinel meaning "membership is sufficient, no module gate".
var validModules = map[string]bool{
	"public": true, "events": true, "members": true, "finances": true,
	"news": true, "polls": true, "settings": true,
}

type routeEntry struct {
	Method      string
	OperationID string
	Segments    []string // path segments after "/teams/{teamId}"; "{...}" denotes a path parameter
	Module      string
	SelfService bool
}

func main() {
	inPath := "openapi/openapi.yaml"
	outPath := "internal/middleware/rbac_table.gen.go"
	if len(os.Args) > 1 {
		inPath = os.Args[1]
	}
	if len(os.Args) > 2 {
		outPath = os.Args[2]
	}

	if err := run(inPath, outPath); err != nil {
		fmt.Fprintln(os.Stderr, "genrbac:", err)
		os.Exit(1)
	}
}

func run(inPath, outPath string) error {
	raw, err := os.ReadFile(inPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", inPath, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parsing %s: %w", inPath, err)
	}

	entries, err := extractRoutes(&doc)
	if err != nil {
		return err
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].OperationID != entries[j].OperationID {
			return entries[i].OperationID < entries[j].OperationID
		}
		return entries[i].Method < entries[j].Method
	})

	out, err := renderTable(entries)
	if err != nil {
		return err
	}

	if err := os.WriteFile(outPath, out, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}
	return nil
}

// extractRoutes walks the "paths" mapping of the spec document and returns
// one routeEntry per team-scoped operation (a path containing "{teamId}").
// Non-team-scoped operations (no {teamId} segment) are skipped entirely --
// RequirePermission passes them through unconditionally, so they carry no
// RBAC extension and are exempt from the completeness check below.
func extractRoutes(doc *yaml.Node) ([]routeEntry, error) {
	root := doc
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}

	pathsNode := mapValue(root, "paths")
	if pathsNode == nil {
		return nil, errNoPathsMapping
	}

	var entries []routeEntry
	var missing []string

	for i := 0; i+1 < len(pathsNode.Content); i += 2 {
		pathKey := pathsNode.Content[i].Value
		pathItem := pathsNode.Content[i+1]
		if !strings.Contains(pathKey, "{teamId}") {
			continue
		}

		pathEntries, pathMissing, err := extractPathOperations(pathKey, pathItem)
		if err != nil {
			return nil, err
		}
		entries = append(entries, pathEntries...)
		missing = append(missing, pathMissing...)
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("%w:\n  %s", errMissingModule, strings.Join(missing, "\n  "))
	}

	return entries, nil
}

// extractPathOperations extracts one routeEntry per HTTP-method operation
// under a single team-scoped path item. Operations missing x-rbac-module are
// returned via the second (missing-descriptions) slice rather than as an
// immediate error, so extractRoutes can report every gap in the spec at once
// instead of stopping at the first one.
func extractPathOperations(pathKey string, pathItem *yaml.Node) (entries []routeEntry, missing []string, err error) {
	segments := pathSegmentsAfterTeam(pathKey)

	for j := 0; j+1 < len(pathItem.Content); j += 2 {
		methodKey := strings.ToLower(pathItem.Content[j].Value)
		if !isHTTPMethod(methodKey) {
			continue // "parameters", "summary", etc. at the path-item level
		}
		opNode := pathItem.Content[j+1]

		opID := mapValueString(opNode, "operationId")
		if opID == "" {
			return nil, nil, fmt.Errorf("%s %s: %w", strings.ToUpper(methodKey), pathKey, errNoOperationID)
		}

		module := mapValueString(opNode, "x-rbac-module")
		if module == "" {
			missing = append(missing, fmt.Sprintf("%s %s (operationId: %s)", strings.ToUpper(methodKey), pathKey, opID))
			continue
		}
		if !validModules[module] {
			return nil, nil, fmt.Errorf("%s: %q: %w", opID, module, errUnrecognizedModule)
		}

		entries = append(entries, routeEntry{
			Method:      strings.ToUpper(methodKey),
			OperationID: opID,
			Segments:    segments,
			Module:      module,
			SelfService: mapValueBool(opNode, "x-rbac-self-service"),
		})
	}

	return entries, missing, nil
}

// pathSegmentsAfterTeam converts an OpenAPI path template like
// "/teams/{teamId}/events/{eventId}/attendance" into the segments that
// follow "/teams/{teamId}", e.g. ["events", "{eventId}", "attendance"].
func pathSegmentsAfterTeam(pathKey string) []string {
	idx := strings.Index(pathKey, "{teamId}")
	rest := strings.Trim(pathKey[idx+len("{teamId}"):], "/")
	if rest == "" {
		return nil
	}
	return strings.Split(rest, "/")
}

func isHTTPMethod(key string) bool {
	for _, m := range httpMethods {
		if key == m {
			return true
		}
	}
	return false
}

// mapValue returns the value node for key in a YAML mapping node, or nil.
func mapValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func mapValueString(node *yaml.Node, key string) string {
	v := mapValue(node, key)
	if v == nil {
		return ""
	}
	return v.Value
}

func mapValueBool(node *yaml.Node, key string) bool {
	v := mapValue(node, key)
	return v != nil && v.Value == "true"
}

const tableTemplate = `// Code generated by cmd/genrbac from openapi/openapi.yaml. DO NOT EDIT.

package middleware

// rbacRouteEntry maps one team-scoped operation to its RBAC module and
// self-service classification, generated from the operation's
// x-rbac-module / x-rbac-self-service extensions.
type rbacRouteEntry struct {
	Method      string
	Segments    []string // path segments after "/teams/{teamId}"; "{...}" is a wildcard
	Module      string
	SelfService bool
}

// rbacRoutes is the complete set of team-scoped operations declared in
// openapi.yaml. A request whose method+path matches no entry here is
// rejected with 404 by RequirePermission, for every HTTP method including
// GET -- see rbac_table.gen.go's generator (cmd/genrbac) for the
// completeness guard that keeps this table exhaustive.
var rbacRoutes = []rbacRouteEntry{
{{- range . }}
	{Method: {{ printf "%q" .Method }}, Segments: {{ segmentsLiteral .Segments }}, Module: {{ printf "%q" .Module }}, SelfService: {{ .SelfService }}}, // {{ .OperationID }}
{{- end }}
}
`

func renderTable(entries []routeEntry) ([]byte, error) {
	tmpl := template.Must(template.New("table").Funcs(template.FuncMap{
		"segmentsLiteral": segmentsLiteral,
	}).Parse(tableTemplate))

	var buf strings.Builder
	if err := tmpl.Execute(&buf, entries); err != nil {
		return nil, fmt.Errorf("rendering table: %w", err)
	}

	formatted, err := format.Source([]byte(buf.String()))
	if err != nil {
		return nil, fmt.Errorf("gofmt generated table: %w", err)
	}
	return formatted, nil
}

func segmentsLiteral(segments []string) string {
	if len(segments) == 0 {
		return "nil"
	}
	quoted := make([]string, len(segments))
	for i, s := range segments {
		quoted[i] = fmt.Sprintf("%q", s)
	}
	return "[]string{" + strings.Join(quoted, ", ") + "}"
}
