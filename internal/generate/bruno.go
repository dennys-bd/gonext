package generate

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

// The .bru templates use [[ ]] so Bruno's own {{baseUrl}} placeholders
// pass through untouched.
var resourceBruTmpl = template.Must(template.New("bru").Delims("[[", "]]").
	ParseFS(resourceTemplates, "templates/resource/*.bru.tmpl"))

var brunoSeqRE = regexp.MustCompile(`(?m)^(\s*seq:\s*)(\d+)\s*$`)

// brunoState is what brunoFolder learned about docs/bruno/<domain>: every
// request's seq, the setup chain's Login (0 when absent) and the
// teardown's file name ("" when absent).
type brunoState struct {
	maxSeq   int
	seqs     map[string]int
	loginSeq int
	teardown string
}

// brunoFolder scans dir's *.bru files; a missing dir is an empty folder.
func brunoFolder(dir, domainPascal string) (brunoState, error) {
	st := brunoState{seqs: map[string]int{}}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return st, nil
	}
	if err != nil {
		return st, fmt.Errorf("reading %s: %w", dir, err)
	}
	login := "Login (" + domainPascal + " Setup).bru"
	teardown := "Logout (" + domainPascal + " Teardown).bru"
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".bru") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return st, fmt.Errorf("reading %s: %w", name, err)
		}
		match := brunoSeqRE.FindSubmatch(data)
		if match == nil {
			continue
		}
		seq, err := strconv.Atoi(string(match[2]))
		if err != nil {
			continue
		}
		st.seqs[name] = seq
		st.maxSeq = max(st.maxSeq, seq)
		switch name {
		case login:
			st.loginSeq = seq
		case teardown:
			st.teardown = name
		}
	}
	return st, nil
}

// renumberSeq rewrites the seq line of the request at p in place.
func renumberSeq(p string, seq int) error {
	data, err := os.ReadFile(p)
	if err != nil {
		return fmt.Errorf("reading %s: %w", filepath.Base(p), err)
	}
	loc := brunoSeqRE.FindSubmatchIndex(data)
	if loc == nil {
		return fmt.Errorf("renumbering %s: no seq line", filepath.Base(p))
	}
	var out []byte
	out = append(out, data[:loc[4]]...)
	out = append(out, strconv.Itoa(seq)...)
	out = append(out, data[loc[5]:]...)
	if err := os.WriteFile(p, out, fileMode); err != nil {
		return fmt.Errorf("writing %s: %w", filepath.Base(p), err)
	}
	return nil
}

// brunoTarget is one request to write: its name (without .bru), the
// template rendering it, the operation it belongs to and its seq.
type brunoTarget struct {
	name, template, op string
	seq                int
}

// brunoPlan is what a resource does to docs/bruno/<domain>: the new
// requests in listing order and the existing files whose seq changes.
type brunoPlan struct {
	targets  []brunoTarget
	renumber map[string]int
}

// planBruno assigns seqs so the run stays valid: every (No Session)
// case precedes Login, since Bruno's CLI jar attaches the session
// cookie to everything after it; new requests continue after the
// folder's maximum; a teardown stays last.
func planBruno(m resourceModel, st brunoState) brunoPlan {
	var noSession, rest []brunoTarget
	for _, op := range AllOps {
		if !m.Op.has(op) {
			continue
		}
		if m.Secured.has(op) {
			noSession = append(noSession, brunoTarget{name: opRequest(m, op) + " (No Session)", template: "no-session.bru.tmpl", op: op})
		}
		rest = append(rest, opTargets(m, op)...)
	}

	plan := brunoPlan{renumber: map[string]int{}}
	next := st.maxSeq + 1
	var chain []brunoTarget
	switch {
	case !m.AnySecured:
	case st.loginSeq == 0:
		next = assignSeqs(noSession, next)
		chain = []brunoTarget{
			{name: "Register (" + m.DomainPascal + " Setup)", template: "setup-register.bru.tmpl"},
			{name: "Confirm Email (" + m.DomainPascal + " Setup)", template: "setup-confirm.bru.tmpl"},
			{name: "Login (" + m.DomainPascal + " Setup)", template: "setup-login.bru.tmpl"},
		}
		next = assignSeqs(chain, next)
	default:
		shift := len(noSession)
		for name, seq := range st.seqs {
			if seq >= st.loginSeq {
				plan.renumber[name] = seq + shift
			}
		}
		assignSeqs(noSession, st.loginSeq)
		next = st.maxSeq + shift + 1
	}
	next = assignSeqs(rest, next)

	// Listing order: chain, then each op's cases, then the teardown.
	plan.targets = append(plan.targets, chain...)
	for _, op := range AllOps {
		for _, t := range noSession {
			if t.op == op {
				plan.targets = append(plan.targets, t)
			}
		}
		for _, t := range rest {
			if t.op == op {
				plan.targets = append(plan.targets, t)
			}
		}
	}
	switch {
	case st.teardown != "":
		plan.renumber[st.teardown] = next
	case len(chain) > 0:
		plan.targets = append(plan.targets, brunoTarget{name: "Logout (" + m.DomainPascal + " Teardown)", template: "teardown-logout.bru.tmpl", seq: next})
	}
	return plan
}

func assignSeqs(targets []brunoTarget, from int) int {
	for i := range targets {
		targets[i].seq = from
		from++
	}
	return from
}

// opRequest names an operation's request: "Create Product", "List Products".
func opRequest(m resourceModel, op string) string {
	if op == "list" {
		return "List " + m.PluralTitle
	}
	return strings.ToUpper(op[:1]) + op[1:] + " " + m.Title
}

// opTargets lists an operation's happy path and error cases; delete's
// happy path goes last so the earlier requests still see the row.
func opTargets(m resourceModel, op string) []brunoTarget {
	name := opRequest(m, op)
	switch op {
	case "create":
		targets := []brunoTarget{{name: name, template: "create.bru.tmpl", op: op}}
		if m.HasFields {
			targets = append(targets, brunoTarget{name: name + " (Missing Field)", template: "create-missing-field.bru.tmpl", op: op})
		}
		return targets
	case "get":
		return []brunoTarget{
			{name: name, template: "get.bru.tmpl", op: op},
			{name: name + " (Not Found)", template: "get-not-found.bru.tmpl", op: op},
		}
	case "list":
		return []brunoTarget{{name: name, template: "list.bru.tmpl", op: op}}
	case "update":
		return []brunoTarget{
			{name: name, template: "update.bru.tmpl", op: op},
			{name: name + " (Not Found)", template: "update-not-found.bru.tmpl", op: op},
		}
	default:
		return []brunoTarget{
			{name: name + " (Not Found)", template: "delete-not-found.bru.tmpl", op: op},
			{name: name, template: "delete.bru.tmpl", op: op},
		}
	}
}

// brunoData is what a .bru template renders: the model plus the
// request's own name, seq and operation.
type brunoData struct {
	resourceModel
	Request   string
	Seq       int
	Operation string
}

// renderBruno renders targets into docs/bruno/<domain>/<name>.bru files.
func renderBruno(m resourceModel, targets []brunoTarget) ([]resourceFile, error) {
	files := make([]resourceFile, 0, len(targets))
	for _, t := range targets {
		rel := path.Join("docs", "bruno", m.Domain, t.name+".bru")
		var buf bytes.Buffer
		data := brunoData{resourceModel: m, Request: t.name, Seq: t.seq, Operation: t.op}
		if err := resourceBruTmpl.ExecuteTemplate(&buf, t.template, data); err != nil {
			return nil, fmt.Errorf("rendering %s: %w", rel, err)
		}
		files = append(files, resourceFile{rel: rel, data: buf.Bytes()})
	}
	return files, nil
}
