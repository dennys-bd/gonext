package generate

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/dennys-bd/gonext/internal/openapi"
)

// Kebab-case ids like get-stub: what httpx registers and what camelCase
// and pascalCase turn into identifiers; also safe as a filename.
var operationIDRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

// pathParamRE finds every {name} placeholder in an operation's path.
var pathParamRE = regexp.MustCompile(`\{([^}]+)\}`)

// pageParam is one route parameter, static across every operation
// generated for the same route.
type pageParam struct {
	Name     string
	TSType   string // string | string[] | string[] | undefined
	PathExpr string // Name, or Name.join("/") for a catch-all
	CatchAll bool
	Used     bool
}

// pageInput is one form input built from a request body property.
type pageInput struct {
	Name      string
	Label     string
	Component string // TextInput | PasswordInput | NumberInput | Checkbox
	InputType string // "email" for <TextInput type="email">, else ""
	Coercion  string // the action's formData -> body value expression
	Sample    string // value the test's FormData helper sets, e.g. "name value", "1", "on"
	Expected  string // the value expected in the test's asserted body object
	Required  bool
	Todo      bool
}

// pageField is one response property rendered in a <dl> or Mantine
// state block.
type pageField struct {
	Name      string
	IsString  bool
	IsBoolean bool
	Todo      bool
	Sample    string // literal TS source for the mocked response value
}

// pageModel is the view model rendered by the page templates, built
// entirely by buildPageModel so templates never branch on the schema.
type pageModel struct {
	ComponentName string
	FormName      string
	FormFile      string
	FormImport    string // FormFile without its extension, for the page's import specifier
	ActionName    string
	StateName     string
	Title         string
	Tag           string
	CallName      string // e.g. api.example.getStub

	// TestPathArg and TestCallWith are precomputed test-file literals
	// so the *.test.ts(x) templates never re-derive them.
	TestPathArg  string // e.g. `{ id: "example" }`, empty when !HasUsed
	TestCallWith string // the full toHaveBeenCalledWith(...) argument, empty for a no-arg call

	IsGet     bool
	HasUsed   bool // the operation uses at least one of the route's params
	HasBody   bool
	HasData   bool // ResponseKind != "none"
	HasDetail bool

	Params     []pageParam // every param the route declares
	UsedParams []pageParam // the subset the operation's path uses, in route order

	Inputs []pageInput
	Fields []pageField

	ResponseKind string // "object" | "value" | "none"

	MantineImports []string
	Warnings       []string
}

// buildPageModel validates op against r and assembles the view model
// Page renders templates from. Each returned error corresponds to one
// row of the design's error table.
func buildPageModel(doc *openapi.Document, op *openapi.Operation, r parsedRoute) (pageModel, error) {
	if err := checkNames(op); err != nil {
		return pageModel{}, err
	}
	if err := checkMethod(op); err != nil {
		return pageModel{}, err
	}
	usedNames, err := checkPathParams(op, r)
	if err != nil {
		return pageModel{}, err
	}
	if err := checkRequiredParams(op); err != nil {
		return pageModel{}, err
	}
	if err := checkBodyContentType(op); err != nil {
		return pageModel{}, err
	}

	m := pageModel{
		ComponentName: pascalCase(op.OperationID) + "Page",
		FormName:      pascalCase(op.OperationID) + "Form",
		FormFile:      op.OperationID + "-form.tsx",
		FormImport:    op.OperationID + "-form",
		ActionName:    camelCase(op.OperationID),
		StateName:     pascalCase(op.OperationID) + "State",
		Tag:           tagOf(op),
		IsGet:         op.Method == "GET",
	}
	m.CallName = callName(op)
	m.Title = op.Summary
	if m.Title == "" {
		m.Title = humanize(op.OperationID)
	} else if isJSXUnsafe(m.Title) {
		fallback := humanize(op.OperationID)
		m.Warnings = append(m.Warnings, fmt.Sprintf("summary of %q contains JSX-unsafe characters; using %q", m.Title, fallback))
		m.Title = fallback
	}

	buildParams(&m, r, usedNames)

	var warnings []string
	if op.RequestBody != nil {
		bodySchema, err := doc.Resolve(op.RequestBody.Content["application/json"].Schema)
		if err != nil {
			return pageModel{}, err
		}
		m.HasBody = true
		m.Inputs, warnings = buildInputs(bodySchema)
		m.Warnings = append(m.Warnings, warnings...)
	}

	kind, fields, respWarnings, err := buildResponse(doc, op)
	if err != nil {
		return pageModel{}, err
	}
	m.ResponseKind = kind
	m.Fields = fields
	m.HasData = kind != "none"
	m.Warnings = append(m.Warnings, respWarnings...)

	m.HasDetail = detailInDefaultResponse(doc, op)
	m.MantineImports = mantineImports(&m)
	buildTestLiterals(&m)

	return m, nil
}

// buildTestLiterals fills TestPathArg and TestCallWith. Every test
// double for a path parameter is "example", but the literal handed to
// the action (TestPathArg) must match the param's own TS type — an
// array for a catch-all — while the mocked SDK call assertion always
// sees the joined scalar the action itself produces.
func buildTestLiterals(m *pageModel) {
	var assertFields []string
	if m.HasUsed {
		var argFields []string
		for _, p := range m.UsedParams {
			argFields = append(argFields, fmt.Sprintf("%s: %s", p.Name, testPathArgLiteral(p)))
			assertFields = append(assertFields, fmt.Sprintf("%s: %q", p.Name, "example"))
		}
		m.TestPathArg = "{ " + strings.Join(argFields, ", ") + " }"
	}

	var opts []string
	if m.HasUsed {
		opts = append(opts, "path: { "+strings.Join(assertFields, ", ")+" }")
	}
	if m.HasBody {
		var props []string
		for _, in := range m.Inputs {
			props = append(props, fmt.Sprintf("%s: %s", in.Name, in.Expected))
		}
		opts = append(opts, "body: { "+strings.Join(props, ", ")+" }")
	}
	if len(opts) > 0 {
		m.TestCallWith = "{ " + strings.Join(opts, ", ") + " }"
	}
}

// isJSXUnsafe reports whether s cannot be spliced verbatim into the
// page templates: a title lands both in JSX text/attribute position
// (unsafe: { } < >) and in a JS string literal in the *.test.tsx
// templates (unsafe: " \), plus raw control characters.
func isJSXUnsafe(s string) bool {
	if strings.ContainsAny(s, "{}<>\"\\") {
		return true
	}
	for _, r := range s {
		if r < 0x20 {
			return true
		}
	}
	return false
}

// checkNames rejects an operationId or tag that would not survive as a
// bare identifier, import path or filename in the generated code.
func checkNames(op *openapi.Operation) error {
	if !operationIDRE.MatchString(op.OperationID) {
		return fmt.Errorf("unsupported: operation id %q is not an identifier", op.OperationID)
	}
	if len(op.Tags) > 0 && !paramNameRE.MatchString(strings.ToLower(op.Tags[0])) {
		return fmt.Errorf("unsupported: operation %q has tag %q; tags must be identifiers", op.OperationID, op.Tags[0])
	}
	return nil
}

func checkMethod(op *openapi.Operation) error {
	switch op.Method {
	case "GET", "POST", "PUT", "PATCH", "DELETE":
		return nil
	default:
		return fmt.Errorf("unsupported method %s for operation %q", op.Method, op.OperationID)
	}
}

// checkPathParams verifies every {name} in op.Path is satisfiable by a
// dynamic or catch-all segment of r, returning the set of names used.
func checkPathParams(op *openapi.Operation, r parsedRoute) (map[string]bool, error) {
	byName := make(map[string]segment, len(r.params()))
	for _, s := range r.params() {
		byName[s.param] = s
	}

	used := make(map[string]bool)
	for _, m := range pathParamRE.FindAllStringSubmatch(op.Path, -1) {
		name := m[1]
		if _, ok := byName[name]; !ok {
			return nil, fmt.Errorf("operation %q needs path parameter %q: add an [%s] segment to the route", op.OperationID, name, name)
		}
		used[name] = true
	}
	return used, nil
}

func checkRequiredParams(op *openapi.Operation) error {
	for _, p := range op.Parameters {
		if p.In != "path" && p.Required {
			return fmt.Errorf("unsupported: operation %q has a required %s parameter %q", op.OperationID, p.In, p.Name)
		}
	}
	return nil
}

func checkBodyContentType(op *openapi.Operation) error {
	if op.RequestBody == nil {
		return nil
	}
	if _, ok := op.RequestBody.Content["application/json"]; !ok {
		return fmt.Errorf("unsupported: operation %q has no application/json request body", op.OperationID)
	}
	return nil
}

func callName(op *openapi.Operation) string {
	return "api." + tagOf(op) + "." + camelCase(op.OperationID)
}

func tagOf(op *openapi.Operation) string {
	if len(op.Tags) == 0 {
		return "default"
	}
	return strings.ToLower(op.Tags[0])
}

// buildParams fills m.Params (every route param), m.UsedParams (the
// subset the operation's path uses), and HasUsed.
func buildParams(m *pageModel, r parsedRoute, used map[string]bool) {
	for _, s := range r.allParams() {
		m.Params = append(m.Params, toPageParam(s, used[s.param]))
	}
	for _, p := range m.Params {
		if p.Used {
			m.UsedParams = append(m.UsedParams, p)
			m.HasUsed = true
		}
	}
}

func toPageParam(s segment, used bool) pageParam {
	p := pageParam{Name: s.param, Used: used}
	switch s.kind {
	case segCatchAll:
		p.TSType = "string[]"
		p.PathExpr = p.Name + `.join("/")`
		p.CatchAll = true
	case segOptionalCatchAll:
		p.TSType = "string[] | undefined"
		p.PathExpr = p.Name + `.join("/")`
		p.CatchAll = true
	default: // segDynamic
		p.TSType = "string"
		p.PathExpr = p.Name
	}
	return p
}

// testPathArgLiteral is the literal a test passes to the action for
// p: an array for a catch-all, matching its TS type, else a scalar.
func testPathArgLiteral(p pageParam) string {
	if p.CatchAll {
		return `["example"]`
	}
	return strconv.Quote("example")
}

// buildInputs maps a resolved request body schema's top-level
// properties to form inputs, in alphabetical order. A freeform object
// (type object, no properties, no $ref) is omitted entirely: hey-ai's
// generated client type carries no such field, so a literal for it
// would be an excess-property error.
func buildInputs(schema *openapi.Schema) ([]pageInput, []string) {
	if schema == nil {
		return nil, nil
	}
	var inputs []pageInput
	var warnings []string
	numbers := 0
	for _, name := range sortedKeys(schema.Properties) {
		if name == "$schema" {
			continue
		}
		if !paramNameRE.MatchString(name) {
			warnings = append(warnings, fmt.Sprintf("skipped property %q: not a valid identifier", name))
			continue
		}
		prop := schema.Properties[name]
		if prop != nil && prop.ReadOnly {
			continue
		}
		required := contains(schema.Required, name)
		if !isPrimitive(prop) {
			warnings = append(warnings, fmt.Sprintf("skipped non-primitive property %q", name))
			if isFreeformObject(prop) {
				continue
			}
			inputs = append(inputs, todoInput(name, prop))
			continue
		}
		if prop.Type == "integer" || prop.Type == "number" {
			numbers++
		}
		inputs = append(inputs, primitiveInput(name, prop, required, numbers))
	}
	return inputs, warnings
}

// isPrimitive reports whether s is a scalar the generator renders an
// input/field for. A nil schema (a malformed `properties: { foo: }`
// entry) is treated as non-primitive.
func isPrimitive(s *openapi.Schema) bool {
	if s == nil || s.Ref != "" {
		return false
	}
	switch s.Type {
	case "string", "integer", "number", "boolean":
		return true
	default:
		return false
	}
}

// isFreeformObject reports whether prop is a `type: object` schema
// with no properties and no $ref — hey-api omits such a property from
// the generated request body type entirely.
func isFreeformObject(prop *openapi.Schema) bool {
	return prop != nil && prop.Type == "object" && prop.Ref == "" && len(prop.Properties) == 0
}

func todoInput(name string, prop *openapi.Schema) pageInput {
	coercion := "{} as never"
	if prop != nil && prop.Type == "array" {
		coercion = "[]"
	}
	return pageInput{Name: name, Label: humanize(name), Coercion: coercion, Expected: coercion, Todo: true}
}

func primitiveInput(name string, prop *openapi.Schema, required bool, number int) pageInput {
	in := pageInput{Name: name, Label: humanize(name), Required: required}
	getExpr := fmt.Sprintf("formData.get(%q)", name)

	switch prop.Type {
	case "string":
		in.Component = "TextInput"
		if prop.Format == "email" || name == "email" {
			in.InputType = "email"
		}
		if prop.Format == "password" || name == "password" {
			in.Component = "PasswordInput"
		}
		in.Coercion = fmt.Sprintf("String(%s ?? \"\")", getExpr)
		if !required {
			in.Coercion += ` || undefined`
		}
		in.Sample = name + " value"
		in.Expected = strconv.Quote(in.Sample)
	case "integer", "number":
		in.Component = "NumberInput"
		if required {
			in.Coercion = fmt.Sprintf("Number(%s)", getExpr)
		} else {
			in.Coercion = fmt.Sprintf("%s ? Number(%s) : undefined", getExpr, getExpr)
		}
		in.Sample = strconv.Itoa(number)
		in.Expected = in.Sample
	case "boolean":
		in.Component = "Checkbox"
		in.Coercion = fmt.Sprintf("%s === \"on\"", getExpr)
		in.Sample = "on"
		in.Expected = "true"
	}
	return in
}

// buildResponse resolves the operation's first 2xx response and
// classifies it into the view model's Fields and ResponseKind.
func buildResponse(doc *openapi.Document, op *openapi.Operation) (kind string, fields []pageField, warnings []string, err error) {
	resp, ok := firstResponse(op.Responses, func(status string) bool { return strings.HasPrefix(status, "2") })
	if !ok || len(resp.Content) == 0 {
		return "none", nil, nil, nil
	}
	media, ok := resp.Content["application/json"]
	if !ok {
		return "none", nil, nil, nil
	}
	schema, err := doc.Resolve(media.Schema)
	if err != nil {
		return "", nil, nil, err
	}
	if schema == nil || schema.Type != "object" {
		return "value", nil, nil, nil
	}

	numbers := 0
	for _, name := range sortedKeys(schema.Properties) {
		if name == "$schema" {
			continue
		}
		if !paramNameRE.MatchString(name) {
			warnings = append(warnings, fmt.Sprintf("skipped property %q: not a valid identifier", name))
			continue
		}
		prop := schema.Properties[name]
		if !isPrimitive(prop) {
			fields = append(fields, pageField{Name: name, Todo: true, Sample: todoSample(prop)})
			warnings = append(warnings, fmt.Sprintf("skipped non-primitive property %q", name))
			continue
		}
		f := pageField{Name: name}
		switch prop.Type {
		case "string":
			f.IsString = true
			f.Sample = strconv.Quote(name + " value")
		case "integer", "number":
			numbers++
			f.Sample = strconv.Itoa(numbers)
		case "boolean":
			f.IsBoolean = true
			f.Sample = "true"
		}
		fields = append(fields, f)
	}
	return "object", fields, warnings, nil
}

func todoSample(prop *openapi.Schema) string {
	if prop != nil && prop.Type == "array" {
		return "[]"
	}
	return "{} as never"
}

// firstResponse returns the response whose status code, sorted,
// first satisfies match.
func firstResponse(responses map[string]openapi.Response, match func(string) bool) (openapi.Response, bool) {
	for _, status := range sortedKeys(responses) {
		if match(status) {
			return responses[status], true
		}
	}
	return openapi.Response{}, false
}

// detailInDefaultResponse reports whether op's default (else first
// 4xx/5xx) response's first media type resolves to a schema with a
// "detail" property, regardless of that media type's name.
func detailInDefaultResponse(doc *openapi.Document, op *openapi.Operation) bool {
	resp, ok := op.Responses["default"]
	if !ok {
		resp, ok = firstResponse(op.Responses, func(status string) bool {
			return strings.HasPrefix(status, "4") || strings.HasPrefix(status, "5")
		})
	}
	if !ok || len(resp.Content) == 0 {
		return false
	}
	mediaTypes := make([]string, 0, len(resp.Content))
	for mt := range resp.Content {
		mediaTypes = append(mediaTypes, mt)
	}
	sort.Strings(mediaTypes)

	schema, err := doc.Resolve(resp.Content[mediaTypes[0]].Schema)
	if err != nil || schema == nil {
		return false
	}
	_, ok = schema.Properties["detail"]
	return ok
}

// mantineImports returns the sorted set of @mantine/core components
// m's templates reference.
func mantineImports(m *pageModel) []string {
	set := map[string]bool{}
	if m.IsGet {
		set["Title"] = true
		set["Text"] = true
	} else {
		set["Button"] = true
		set["Stack"] = true
		set["Text"] = true
		for _, in := range m.Inputs {
			if in.Component != "" {
				set[in.Component] = true
			}
		}
	}
	out := make([]string, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// splitWords splits a kebab-case, snake_case or camelCase identifier
// into its lower-cased component words.
func splitWords(s string) []string {
	var words []string
	var cur strings.Builder
	runes := []rune(s)
	flush := func() {
		if cur.Len() > 0 {
			words = append(words, cur.String())
			cur.Reset()
		}
	}
	for i, r := range runes {
		switch {
		case r == '-' || r == '_' || r == ' ':
			flush()
		case i > 0 && unicode.IsUpper(r) && !unicode.IsUpper(runes[i-1]):
			flush()
			cur.WriteRune(r)
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return words
}

// camelCase converts an operationId such as "get-current-user" to
// "getCurrentUser".
func camelCase(s string) string {
	p := pascalCase(s)
	if p == "" {
		return p
	}
	return strings.ToLower(p[:1]) + p[1:]
}

// pascalCase converts an operationId such as "get-current-user" to
// "GetCurrentUser".
func pascalCase(s string) string {
	var b strings.Builder
	for _, w := range splitWords(s) {
		lower := strings.ToLower(w)
		b.WriteString(strings.ToUpper(lower[:1]) + lower[1:])
	}
	return b.String()
}

// humanize converts an identifier such as "createdAt" or "get-stub"
// into a sentence-cased label: "Created at", "Get stub".
func humanize(s string) string {
	words := splitWords(s)
	for i, w := range words {
		words[i] = strings.ToLower(w)
	}
	if len(words) > 0 {
		words[0] = strings.ToUpper(words[0][:1]) + words[0][1:]
	}
	return strings.Join(words, " ")
}
