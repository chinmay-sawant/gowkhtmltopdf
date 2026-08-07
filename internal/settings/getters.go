package settings

import (
	"strconv"
	"strings"
)

// Get methods use the Get half of the field-descriptor tables built in
// reflect.go (same key names as Set — registered once per type).

// Get reads a dotted global key as its canonical string form. ok is false
// for unknown keys. Accepted ignored keys return the last Set value.
func (g *PdfGlobal) Get(name string) (string, bool) {
	return getForKey(g, globalKeys, &g.Ignored, name)
}

// Get reads a dotted object key as its canonical string form.
func (o *PdfObject) Get(name string) (string, bool) {
	return getForKey(o, objectKeys, &o.Ignored, name)
}

// Get reads a dotted image-mode key as its canonical string form.
func (g *ImageGlobal) Get(name string) (string, bool) {
	return getForKey(g, imageKeys, &g.Ignored, name)
}

func fmtBool(b bool) string { return strconv.FormatBool(b) }

func fmtFloat(f float64) string {
	return strconv.FormatFloat(f, 'g', -1, 64)
}

func fmtInt(n int) string { return strconv.Itoa(n) }

func fmtStrings(ss []string) string {
	return strings.Join(ss, "\n")
}
