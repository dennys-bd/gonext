package generate

import (
	"fmt"
	"go/token"
	"regexp"
	"slices"
	"strings"
)

// ResourceSpec is what `gonext generate resource` asks for. Plural and
// Auth empty mean derive and "all"; Ops nil means every operation.
type ResourceSpec struct {
	Domain, Name, Plural, Auth string
	Fields, Ops                []string
}

// AllOps lists every resource operation in canonical order.
var AllOps = []string{"create", "get", "list", "update", "delete"}

var resourceNameRE = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// fieldType is one row of the field grammar: how a token type renders in
// Go, SQL, Huma tags and the sample literals the generated tests use.
type fieldType struct {
	Token, Go, Column, HumaTags, JSONFormat    string
	GoSample, GoUpdated, BruSample, BruUpdated string
	ImportsTime                                bool
}

var fieldTypes = map[string]fieldType{
	"string": {Token: "string", Go: "string", Column: "text NOT NULL", HumaTags: `minLength:"1"`,
		GoSample: `"demo"`, GoUpdated: `"demo-updated"`, BruSample: `"demo"`, BruUpdated: `"demo-updated"`},
	"int": {Token: "int", Go: "int", Column: "integer NOT NULL",
		GoSample: "42", GoUpdated: "43", BruSample: "42", BruUpdated: "43"},
	"int64": {Token: "int64", Go: "int64", Column: "bigint NOT NULL", JSONFormat: "int64",
		GoSample: "int64(42)", GoUpdated: "int64(43)", BruSample: "42", BruUpdated: "43"},
	"float": {Token: "float", Go: "float64", Column: "double precision NOT NULL",
		GoSample: "9.5", GoUpdated: "10.5", BruSample: "9.5", BruUpdated: "10.5"},
	"bool": {Token: "bool", Go: "bool", Column: "boolean NOT NULL",
		GoSample: "true", GoUpdated: "false", BruSample: "true", BruUpdated: "false"},
	"time": {Token: "time", Go: "time.Time", Column: "timestamptz NOT NULL", JSONFormat: "date-time",
		GoSample: "time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)", GoUpdated: "time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)",
		BruSample: `"2026-01-02T03:04:05Z"`, BruUpdated: `"2026-02-03T04:05:06Z"`, ImportsTime: true},
}

const fieldTypeList = "string, int, int64, float, bool or time"
const opList = "create, get, list, update or delete"

var implicitFields = []string{"id", "created_at", "updated_at"}

// reservedFields are the receiver and locals the generated service
// declares next to the positional field parameters.
var reservedFields = []string{"ctx", "id", "now", "err", "s"}

type resourceField struct {
	Name, Pascal, Camel, Column string
	Type                        fieldType
	// Mismatch and MismatchUpdated are printf formats taking the entity
	// variable, rendering the test's "field differs from the sample"
	// condition in the form each type compares by (see mismatch).
	Mismatch, MismatchUpdated string
	// BruExpect and BruExpectUpdated are the Bruno assertions that the
	// response body carries the sample (see bruExpect).
	BruExpect, BruExpectUpdated string
}

// opSet flags a subset of AllOps; used for the selected operations and
// again for the secured ones.
type opSet struct {
	Create, Get, List, Update, Delete bool
}

func (s opSet) has(op string) bool {
	switch op {
	case "create":
		return s.Create
	case "get":
		return s.Get
	case "list":
		return s.List
	case "update":
		return s.Update
	case "delete":
		return s.Delete
	}
	return false
}

func (s opSet) with(op string) opSet {
	switch op {
	case "create":
		s.Create = true
	case "get":
		s.Get = true
	case "list":
		s.List = true
	case "update":
		s.Update = true
	case "delete":
		s.Delete = true
	}
	return s
}

// resourceModel is everything the resource templates read; every string
// is precomputed here so the templates call no funcs.
type resourceModel struct {
	Domain, DomainPascal, DomainCamel  string
	Module                             string
	Name, Pascal, Camel, Human, Title  string
	Plural, PluralPascal, PluralCamel  string
	PluralHuman, PluralTitle           string
	Table, Route                       string
	SessionVar, IDVar                  string
	OpID                               map[string]string
	BodySample, BodyUpdated            string
	Fields                             []resourceField
	ColumnWidth                        int
	HasFields, ImportsTime, AnySecured bool
	Op, Secured                        opSet
	NewPackages                        map[string]bool
}

func parseFields(tokens []string) ([]resourceField, error) {
	var fields []resourceField
	seen := map[string]bool{}
	for _, tok := range tokens {
		name, typ, ok := strings.Cut(tok, ":")
		if !ok || name == "" || typ == "" {
			return nil, fmt.Errorf("invalid field %q: use name:type", tok)
		}
		if !resourceNameRE.MatchString(name) {
			return nil, fmt.Errorf("invalid field name %q: use snake_case", name)
		}
		if slices.Contains(implicitFields, name) {
			return nil, fmt.Errorf("field %q is implicit", name)
		}
		if token.IsKeyword(name) {
			return nil, fmt.Errorf("field %q is a Go keyword", name)
		}
		if slices.Contains(reservedFields, name) {
			return nil, fmt.Errorf("field %q is reserved", name)
		}
		if seen[name] {
			return nil, fmt.Errorf("field %q repeated", name)
		}
		ft, ok := fieldTypes[typ]
		if !ok {
			return nil, fmt.Errorf("unknown type %q for field %q: use %s", typ, name, fieldTypeList)
		}
		seen[name] = true
		fields = append(fields, resourceField{
			Name:   name,
			Pascal: pascalCase(name),
			Camel:  camelCase(name),
			Column: name,
			Type:   ft,
		})
	}
	return fields, nil
}

// pluralize applies the spec's rule: y after a consonant → ies; s, x, z,
// ch, sh → es; otherwise +s.
func pluralize(name string) string {
	n := len(name)
	switch {
	case n >= 2 && name[n-1] == 'y' && !strings.ContainsRune("aeiou", rune(name[n-2])):
		return name[:n-1] + "ies"
	case strings.HasSuffix(name, "s"), strings.HasSuffix(name, "x"), strings.HasSuffix(name, "z"),
		strings.HasSuffix(name, "ch"), strings.HasSuffix(name, "sh"):
		return name + "es"
	default:
		return name + "s"
	}
}

// parseOps validates ops and returns them deduplicated in canonical
// order; nil or empty selects every operation.
func parseOps(ops []string) ([]string, error) {
	if len(ops) == 0 {
		return slices.Clone(AllOps), nil
	}
	for _, op := range ops {
		if !slices.Contains(AllOps, op) {
			return nil, fmt.Errorf("unknown operation %q: use %s", op, opList)
		}
	}
	var out []string
	for _, op := range AllOps {
		if slices.Contains(ops, op) {
			out = append(out, op)
		}
	}
	return out, nil
}

// securedOps applies an --auth policy to the selected ops; an empty
// policy means "all".
func securedOps(policy string, ops []string) (opSet, error) {
	if policy == "" {
		policy = "all"
	}
	if !slices.Contains([]string{"all", "mutations", "public"}, policy) {
		return opSet{}, fmt.Errorf("unknown auth policy %q: use all, mutations or public", policy)
	}
	var secured opSet
	for _, op := range ops {
		isRead := op == "get" || op == "list"
		if policy == "all" || (policy == "mutations" && !isRead) {
			secured = secured.with(op)
		}
	}
	return secured, nil
}

func buildResourceModel(module string, spec ResourceSpec) (resourceModel, error) {
	if !resourceNameRE.MatchString(spec.Name) {
		return resourceModel{}, fmt.Errorf("invalid resource name %q: use snake_case", spec.Name)
	}
	fields, err := parseFields(spec.Fields)
	if err != nil {
		return resourceModel{}, err
	}
	// The entity's local in the service shares its scope with the
	// positional field parameters.
	for _, f := range fields {
		if f.Name == spec.Name {
			return resourceModel{}, fmt.Errorf("field %q is reserved", f.Name)
		}
	}
	plural := spec.Plural
	if plural == "" {
		plural = pluralize(spec.Name)
	} else if !resourceNameRE.MatchString(plural) {
		return resourceModel{}, fmt.Errorf("invalid plural %q: use snake_case", plural)
	}
	ops, err := parseOps(spec.Ops)
	if err != nil {
		return resourceModel{}, err
	}
	secured, err := securedOps(spec.Auth, ops)
	if err != nil {
		return resourceModel{}, err
	}

	m := resourceModel{
		Domain:       spec.Domain,
		DomainPascal: pascalCase(spec.Domain),
		DomainCamel:  camelCase(spec.Domain),
		Module:       module,
		Name:         spec.Name,
		Pascal:       pascalCase(spec.Name),
		Camel:        camelCase(spec.Name),
		Human:        strings.ReplaceAll(spec.Name, "_", " "),
		Title:        titleCase(spec.Name),
		Plural:       plural,
		PluralPascal: pascalCase(plural),
		PluralCamel:  camelCase(plural),
		PluralHuman:  strings.ReplaceAll(plural, "_", " "),
		PluralTitle:  titleCase(plural),
		Table:        plural,
		Route:        "/" + plural,
		SessionVar:   camelCase(spec.Domain) + "SessionCookie",
		IDVar:        camelCase(spec.Name) + "Id",
		Fields:       fields,
		ColumnWidth:  len("created_at"),
		HasFields:    len(fields) > 0,
		Secured:      secured,
		AnySecured:   secured != opSet{},
		NewPackages:  map[string]bool{},
	}
	kebabName := strings.ReplaceAll(spec.Name, "_", "-")
	kebabPlural := strings.ReplaceAll(plural, "_", "-")
	m.OpID = map[string]string{
		"create": "create-" + kebabName,
		"get":    "get-" + kebabName,
		"list":   "list-" + kebabPlural,
		"update": "update-" + kebabName,
		"delete": "delete-" + kebabName,
	}
	for _, op := range ops {
		m.Op = m.Op.with(op)
	}
	var sample, updated []string
	for i, f := range fields {
		m.Fields[i].Mismatch = mismatch(f, f.Type.GoSample)
		m.Fields[i].MismatchUpdated = mismatch(f, f.Type.GoUpdated)
		m.Fields[i].BruExpect = bruExpect(f, f.Type.BruSample)
		m.Fields[i].BruExpectUpdated = bruExpect(f, f.Type.BruUpdated)
		m.ImportsTime = m.ImportsTime || f.Type.ImportsTime
		m.ColumnWidth = max(m.ColumnWidth, len(f.Column))
		sample = append(sample, `"`+f.Camel+`": `+f.Type.BruSample)
		updated = append(updated, `"`+f.Camel+`": `+f.Type.BruUpdated)
	}
	// The JSON bodies the presentation test posts, as Go map literals;
	// the Bruno samples are JSON literals that read the same in Go.
	m.BodySample = strings.Join(sample, ", ")
	m.BodyUpdated = strings.Join(updated, ", ")
	return m, nil
}

// titleCase renders a snake_case name as Bruno request words: "order_line"
// → "Order Line".
func titleCase(s string) string {
	words := splitWords(s)
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

// mismatch is the condition that f's value on %s is not literal: bools
// read as their own value (staticcheck rejects "!= true"), times
// compare with Equal, everything else with !=.
func mismatch(f resourceField, literal string) string {
	switch {
	case f.Type.Token == "bool" && literal == "true":
		return "!%s." + f.Pascal
	case f.Type.Token == "bool":
		return "%s." + f.Pascal
	case f.Type.ImportsTime:
		return "!%s." + f.Pascal + ".Equal(" + literal + ")"
	default:
		return "%s." + f.Pascal + " != " + literal
	}
}

// bruExpect asserts that body.<field> equals literal. Times compare as
// instants: Postgres hands a timestamptz back in the server's zone, so
// the string differs from the UTC sample while the instant does not.
func bruExpect(f resourceField, literal string) string {
	if f.Type.ImportsTime {
		return "expect(Date.parse(body." + f.Camel + ")).to.equal(Date.parse(" + literal + "))"
	}
	return "expect(body." + f.Camel + ").to.equal(" + literal + ")"
}
