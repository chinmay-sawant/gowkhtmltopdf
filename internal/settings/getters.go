package settings

import (
	"strconv"
	"strings"
)

// Get methods use the Get half of the unified key tables built in reflect.go
// (same key names as Set — registered once via gReg/oReg/iReg).

// Get reads a dotted global key as its canonical string form. ok is false
// for unknown keys. Accepted ignored keys return the last Set value.
func (g *PdfGlobal) Get(name string) (string, bool) {
	ensureKeyTables()
	key := normalizeDots(name)
	if fn, ok := globalGetTable[key]; ok {
		return fn(g)
	}
	if g.Ignored != nil {
		if v, ok := g.Ignored[key]; ok {
			return v, true
		}
	}
	return "", false
}

// Get reads a dotted object key as its canonical string form.
func (o *PdfObject) Get(name string) (string, bool) {
	ensureKeyTables()
	key := normalizeDots(name)
	if fn, ok := objectGetTable[key]; ok {
		return fn(o)
	}
	if o.Ignored != nil {
		if v, ok := o.Ignored[key]; ok {
			return v, true
		}
	}
	return "", false
}

// Get reads a dotted image-mode key as its canonical string form.
func (g *ImageGlobal) Get(name string) (string, bool) {
	ensureKeyTables()
	key := normalizeDots(name)
	if fn, ok := imageGetTable[key]; ok {
		return fn(g)
	}
	if g.Ignored != nil {
		if v, ok := g.Ignored[key]; ok {
			return v, true
		}
	}
	return "", false
}

func fmtBool(b bool) string { return strconv.FormatBool(b) }

func fmtFloat(f float64) string {
	return strconv.FormatFloat(f, 'g', -1, 64)
}

func fmtInt(n int) string { return strconv.Itoa(n) }

func fmtStrings(ss []string) string {
	return strings.Join(ss, "\n")
}
