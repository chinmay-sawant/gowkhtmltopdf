// Package pdf implements a stdlib-only PDF 1.4 writer: indirect objects,
// xref, catalog/pages tree, content streams (Flate), base-14 fonts, images,
// link annotations, named destinations and outlines. Deterministic output
// for golden tests (creation date is injectable).
package pdf

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	errPDFFinalized = errors.New("pdf: document already finalized")
	errPDFNoPages   = errors.New("pdf: no pages")
)

// Version is the PDF header version emitted.
const Version = "1.4"

// object is one indirect object (pre-serialized dict + optional stream).
type object struct {
	id     int
	dict   string // body text before stream (may be empty)
	stream []byte
}

// objRef is a typed indirect-object handle; the "N 0 R" spelling is a
// formatting concern, not a data type, and refs cannot be malformed.
type objRef int

func (r objRef) String() string { return strconv.Itoa(int(r)) + " 0 R" }

// parseRef parses an "N 0 R" reference string (the exported surface still
// carries refs as strings). ok is false when s is not a valid ref.
func parseRef(s string) (objRef, bool) {
	fields := strings.Fields(s)
	if len(fields) != 3 || fields[1] != "0" || fields[2] != "R" {
		return 0, false
	}

	n, err := strconv.Atoi(fields[0])
	if err != nil || n <= 0 {
		return 0, false
	}

	return objRef(n), true
}

// dict is a tiny ordered builder so PDF syntax (escaping, /Name tokens,
// pdfString folding) lives in one place instead of ~20 fmt.Sprintf sites.
type dict []string

func (d dict) add(k string, v ...string) dict { return append(d, append([]string{k}, v...)...) }

func (d dict) String() string { return "<< " + strings.Join(d, " ") + " >>" }

// Document is a PDF under construction.
type Document struct {
	objects        []*object
	info           map[string]string
	useCompression bool
	grayscale      bool
	creationTime   time.Time // zero value → deterministic fixed date
	nextID         int
	pages          []*Page
	outlineRoot    *Outline
	fontCache      map[string]objRef // subset key -> font dict ref
	catalogRef     objRef            // set by finalize
	infoRef        objRef            // set by finalize
	finalized      bool
}

// NewDocument creates an empty PDF document.
func NewDocument() *Document {
	return &Document{ //nolint:exhaustruct // intentional zero-value fields
		info:           map[string]string{},
		useCompression: true,
		fontCache:      map[string]objRef{},
	}
}

// SetCompression toggles Flate compression of content/image streams.
func (d *Document) SetCompression(on bool) { d.useCompression = on }

// SetGrayscale paints grayscale content (converted at paint time by layout).
func (d *Document) SetGrayscale(on bool) { d.grayscale = on }

// Grayscale reports whether grayscale mode is on.
func (d *Document) Grayscale() bool { return d.grayscale }

// SetCreationTime pins the document CreationDate (deterministic tests).
func (d *Document) SetCreationTime(t time.Time) { d.creationTime = t }

// SetInfo sets an Info-dict entry (Title, Creator, Producer, Subject, …).
func (d *Document) SetInfo(key, value string) { d.info[key] = value }

// newObject allocates an indirect object and returns its typed reference.
func (d *Document) newObject() objRef {
	d.nextID++
	id := d.nextID
	d.objects = append(d.objects, &object{id: id}) //nolint:exhaustruct // intentional zero-value fields

	return objRef(id)
}

// setDict replaces the object's dict body.
func (d *Document) setDict(r objRef, dict string) {
	d.objects[int(r)-1].dict = dict
}

// setStream attaches a raw stream (compressed later at write time).
func (d *Document) setStream(r objRef, raw []byte) {
	d.objects[int(r)-1].stream = raw
}

// Page is one page of the document.
type Page struct {
	doc        *Document
	ref        objRef
	width      float64
	height     float64
	content    *Content
	contentRef objRef
	annots     []annotation
}

// annotation is a link annotation.
type annotation struct {
	rect     [4]float64 // x1,y1,x2,y2 in PDF coords
	uri      string     // external link
	destPage int        // internal link target (0-based page index)
	destX    float64
	destY    float64
	hasDest  bool
	annotRef objRef
}

// AddPage appends a page with the given size in points.
func (d *Document) AddPage(width, height float64) *Page {
	page := &Page{doc: d, width: width, height: height} //nolint:exhaustruct // intentional zero-value fields
	page.ref = d.newObject()
	contentRef := d.newObject()
	page.contentRef = contentRef
	page.content = NewContent()
	page.content.doc = d
	d.pages = append(d.pages, page)

	return page
}

// PageCount returns the number of pages currently in the document.
func (d *Document) PageCount() int { return len(d.pages) }

// PageRef returns the object reference string of the page at index idx, or
// "" when idx is out of range. Page objects are allocated at AddPage time and
// their refs are stable, so this is safe to call after all AddPage calls and
// before Write - e.g. to wire outline destinations and link annotations.
func (d *Document) PageRef(idx int) string {
	if idx < 0 || idx >= len(d.pages) {
		return ""
	}

	return d.pages[idx].ref.String()
}

// PageAt returns the Page at index idx, or nil when out of range. Used to
// attach annotations (AddLinkDest/AddLinkURI) after painting.
func (d *Document) PageAt(idx int) *Page {
	if idx < 0 || idx >= len(d.pages) {
		return nil
	}

	return d.pages[idx]
}

// ReorderPages replaces the page order, e.g. for copies/collate assembly.
// Page objects are self-contained - their /Contents stream and /Annots live
// on the page itself and the pages tree is a flat single-level /Kids list
// built at finalize - so permuting d.pages is sufficient. order must be a
// permutation of the current page indices; anything else is an error.
func (d *Document) ReorderPages(order []int) error {
	if d.finalized {
		return errPDFFinalized
	}

	if len(order) != len(d.pages) {
		//nolint:err113 // dynamic counts in message
		return fmt.Errorf("pdf: reorder: order has %d entries, document has %d pages",
			len(order), len(d.pages))
	}

	seen := make([]bool, len(d.pages))
	next := make([]*Page, len(d.pages))

	for index, idx := range order {
		if idx < 0 || idx >= len(d.pages) {
			//nolint:err113 // dynamic index in message
			return fmt.Errorf("pdf: reorder: index %d out of range (0..%d)", idx, len(d.pages)-1)
		}

		if seen[idx] {
			//nolint:err113 // dynamic index in message
			return fmt.Errorf("pdf: reorder: index %d appears more than once", idx)
		}

		seen[idx] = true
		next[index] = d.pages[idx]
	}

	d.pages = next

	return nil
}

// DuplicatePage appends a fresh page object that paints the same content as
// the page at index i: same size, a new /Contents object with the same
// stream bytes, and independent copies of the link annotations. Parsed fonts
// and already-materialized image objects may be shared by the document, but
// each page owns its resource maps. Used to materialize
// copies/collate page runs before ReorderPages.
func (d *Document) DuplicatePage(idx int) (*Page, error) {
	if d.finalized {
		return nil, errPDFFinalized
	}

	if idx < 0 || idx >= len(d.pages) {
		//nolint:err113 // dynamic index in message
		return nil, fmt.Errorf("pdf: duplicate: page index %d out of range (0..%d)", idx, len(d.pages)-1)
	}

	src := d.pages[idx]
	p := d.AddPage(src.width, src.height)
	p.content = cloneContent(src.content)
	p.annots = append([]annotation(nil), src.annots...)

	return p, nil
}

// Width returns the page width in points.
func (p *Page) Width() float64 { return p.width }

// Height returns the page height in points.
func (p *Page) Height() float64 { return p.height }

// Content returns the page content stream builder.
func (p *Page) Content() *Content { return p.content }

// AddLinkURI adds an external URI annotation.
func (p *Page) AddLinkURI(rect [4]float64, uri string) {
	p.annots = append(p.annots, annotation{rect: rect, uri: uri}) //nolint:exhaustruct // intentional zero-value fields
}

// AddLinkDest adds an internal GoTo annotation to a page (0-based index).
func (p *Page) AddLinkDest(rect [4]float64, page int, x, y float64) {
	p.annots = append(p.annots, annotation{ //nolint:exhaustruct // intentional zero-value fields
		rect:     rect,
		destPage: page,
		destX:    x,
		destY:    y,
		hasDest:  true,
	})
}

// Outline is a PDF outline (bookmark) node.
type Outline struct {
	Title    string
	PageRef  string // page object ref, set by caller after layout
	X, Y     float64
	Children []*Outline

	refStr objRef // assigned during finalize
}

// SetOutline installs the document outline tree.
func (d *Document) SetOutline(root *Outline) { d.outlineRoot = root }

// Write serializes the full PDF to w.
func (d *Document) Write(width io.Writer) error {
	if err := d.finalize(); err != nil {
		return err
	}

	var buf bytes.Buffer

	fmt.Fprintf(&buf, "%%PDF-%s\n", Version)
	fmt.Fprintln(&buf, "%\xe2\xe3\xcf\xd3") // binary comment

	offsets := make([]int64, len(d.objects)+1)

	for _, obj := range d.objects {
		if obj.dict == "" {
			// Object was allocated but never materialized; leave its offset
			// unrecorded so the xref cannot point at the *next* object.
			continue
		}

		offsets[obj.id] = int64(buf.Len())
		fmt.Fprintf(&buf, "%d 0 obj\n", obj.id)
		buf.WriteString(obj.dict)

		if len(obj.stream) > 0 {
			buf.WriteString("\nstream\n")
			buf.Write(obj.stream)
			buf.WriteString("\nendstream")
		}

		fmt.Fprintln(&buf)
		fmt.Fprintln(&buf, "endobj")
	}

	xrefPos := int64(buf.Len())
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(d.objects)+1)
	fmt.Fprintln(&buf, "0000000000 65535 f ")

	for i := 1; i <= len(d.objects); i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}

	fmt.Fprintln(&buf, "trailer")
	fmt.Fprintf(&buf, "<< /Size %d /Root %s /Info %s >>\n", len(d.objects)+1, d.catalogRef, d.infoRef)
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefPos)

	if _, err := width.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("pdf: write: %w", err)
	}

	return nil
}

// WriteTo implements io.WriterTo.
func (d *Document) WriteTo(width io.Writer) (int64, error) {
	var buf bytes.Buffer
	if err := d.Write(&buf); err != nil {
		return 0, err
	}

	if _, err := width.Write(buf.Bytes()); err != nil {
		return 0, fmt.Errorf("pdf: write: %w", err)
	}

	return int64(buf.Len()), nil
}

// finalize builds catalog, pages tree, fonts, images, annots, outlines and
// page objects once.
func (d *Document) finalize() error {
	if d.finalized {
		return nil
	}

	if len(d.pages) == 0 {
		return errPDFNoPages
	}

	catalogRef := d.newObject()
	infoRef := d.newObject()
	pagesRef := d.newObject()

	pageRefs := make([]string, 0, len(d.pages))
	for _, p := range d.pages {
		pageRefs = append(pageRefs, p.ref.String())
	}

	// pages tree root
	d.setDict(pagesRef, fmt.Sprintf(
		"<< /Type /Pages\n/Kids [%s]\n/Count %d >>",
		strings.Join(pageRefs, " "), len(pageRefs)))

	// Outlines must be finalized before the catalog dict is written so
	// outlineRoot.refStr is assigned; otherwise /Outlines is emitted with an
	// empty value and the Catalog dictionary is malformed (viewers show no
	// content / fail to open the file).
	if d.outlineRoot != nil {
		if err := d.finalizeOutlines(d.outlineRoot); err != nil {
			return err
		}
	}

	d.setDict(catalogRef, catalogDict(d.outlineRoot, pagesRef))
	d.setDict(infoRef, d.infoDict())

	// per-page objects
	for _, p := range d.pages {
		if err := d.finalizePage(p, pagesRef); err != nil {
			return err
		}
	}

	d.catalogRef = catalogRef
	d.infoRef = infoRef
	d.finalized = true

	return nil
}

// catalogDict builds the /Catalog dictionary, wiring /Outlines when set.
func catalogDict(outlineRoot *Outline, pagesRef objRef) string {
	cat := dict{}.add("/Type", "/Catalog").add("/Pages", pagesRef.String())
	if outlineRoot != nil {
		cat = cat.add("/Outlines", outlineRoot.refStr.String()).add("/PageMode", "/UseOutlines")
	}

	return cat.String()
}

// infoDict builds the /Info dictionary with the (injectable) timestamps.
func (d *Document) infoDict() string {
	now := d.creationTime
	if now.IsZero() {
		now = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC) // deterministic default
	}

	info := dict{}
	for _, k := range []string{"Title", "Subject", "Author", "Keywords"} {
		if v, ok := d.info[k]; ok && v != "" {
			info = info.add("/"+k, pdfString(v))
		}
	}

	return info.add("/Creator", pdfString(d.info["Creator"])).
		add("/Producer", pdfString("gowkhtmltopdf "+Version)).
		add("/CreationDate", pdfString(pdfDate(now))).
		add("/ModDate", pdfString(pdfDate(now))).
		String()
}

func (d *Document) finalizePage(page *Page, pagesRef objRef) error {
	raw := page.content.Bytes()
	if d.useCompression {
		raw = flateBytes(raw)
		d.setDict(page.contentRef, "<< /Length "+strconv.Itoa(len(raw))+" /Filter /FlateDecode >>")
	} else {
		d.setDict(page.contentRef, "<< /Length "+strconv.Itoa(len(raw))+" >>")
	}

	d.setStream(page.contentRef, raw)

	res, err := buildPageResources(page.content)
	if err != nil {
		return err
	}

	parts := []string{
		"<< /Type /Page",
		"/Parent " + pagesRef.String(),
		fmt.Sprintf("/MediaBox [0 0 %s %s]", num(page.width), num(page.height)),
		"/Resources " + res,
		"/Contents " + page.contentRef.String(),
	}

	if len(page.annots) > 0 {
		d.buildAnnots(page)

		var refs []string
		for _, a := range page.annots {
			refs = append(refs, a.annotRef.String())
		}

		parts = append(parts, "/Annots ["+strings.Join(refs, " ")+"]")
	}

	parts = append(parts, ">>")
	d.setDict(page.ref, strings.Join(parts, "\n"))

	return nil
}

// buildPageResources assembles the /Resources dict from the content's fonts
// and images.
func buildPageResources(content *Content) (string, error) {
	fonts, err := content.fonts()
	if err != nil {
		return "", err
	}

	imgResources := content.imageResources()

	var res strings.Builder

	res.WriteString("<< /ProcSet [/PDF /Text /ImageB /ImageC /ImageI]")

	if len(fonts) > 0 {
		res.WriteString(" /Font <<")

		for name, ref := range fonts {
			res.WriteString(" /" + name + " " + ref)
		}

		res.WriteString(" >>")
	}

	if len(imgResources) > 0 {
		res.WriteString(" /XObject <<")

		for name, ref := range imgResources {
			res.WriteString(" /" + name + " " + ref)
		}

		res.WriteString(" >>")
	}

	if gs := content.extGState(); gs != "" {
		res.WriteString(" " + gs)
	}

	res.WriteString(" >>")

	return res.String(), nil
}

func (d *Document) buildAnnots(p *Page) {
	for i := range p.annots {
		arg := &p.annots[i]
		arg.annotRef = d.newObject()
		r := arg.rect

		var buf strings.Builder

		fmt.Fprintf(&buf, "<< /Type /Annot /Subtype /Link /Rect [%s %s %s %s]",
			num(r[0]), num(r[1]), num(r[2]), num(r[3]))

		if arg.hasDest {
			if arg.destPage >= 0 && arg.destPage < len(d.pages) {
				fmt.Fprintf(&buf, " /Dest [%s /XYZ %s %s null]",
					d.pages[arg.destPage].ref, num(arg.destX), num(arg.destY))
			}
		} else {
			fmt.Fprintf(&buf, " /A << /S /URI /URI %s >>", pdfString(arg.uri))
		}

		buf.WriteString(" >>")
		d.setDict(arg.annotRef, buf.String())
	}
}

// finalizeOutlines allocates outline item refs and serializes the tree.
// The root Outline is the /Outlines dictionary; its children are items.
// PageRef strings are validated as object refs; a bogus ref fails the
// document instead of emitting a corrupt /Dest.
func (d *Document) finalizeOutlines(root *Outline) error {
	assignRefs(root, d)

	if len(root.Children) == 0 {
		d.setDict(root.refStr, "<< /Type /Outlines /Count 0 >>")

		return nil
	}

	first := root.Children[0].refStr
	last := root.Children[len(root.Children)-1].refStr
	d.setDict(root.refStr, fmt.Sprintf(
		"<< /Type /Outlines /First %s /Last %s /Count %d >>",
		first, last, outlineCount(root)))

	return buildOutlineItems(root, root.refStr, d)
}

// buildOutlineItems writes each child item of parent, with /Parent /Prev
// /Next sibling links, recursing into nested children.
func buildOutlineItems(parent *Outline, parentRef objRef, doc *Document) error {
	for idx, child := range parent.Children {
		var parts []string
		parts = append(parts, "<< /Title "+pdfString(child.Title))

		dest, err := outlineDest(child, doc)
		if err != nil {
			return err
		}

		if dest != "" {
			parts = append(parts, dest)
		}

		if len(child.Children) > 0 {
			first := child.Children[0].refStr
			last := child.Children[len(child.Children)-1].refStr
			parts = append(parts, fmt.Sprintf("/First %s /Last %s /Count %d", first, last, countChildren(child)))
		}

		parts = append(parts, "/Parent "+parentRef.String())
		if idx > 0 {
			parts = append(parts, "/Prev "+parent.Children[idx-1].refStr.String())
		}

		if idx < len(parent.Children)-1 {
			parts = append(parts, "/Next "+parent.Children[idx+1].refStr.String())
		}

		parts = append(parts, ">>")
		doc.setDict(child.refStr, strings.Join(parts, " "))

		if len(child.Children) > 0 {
			if err := buildOutlineItems(child, child.refStr, doc); err != nil {
				return err
			}
		}
	}

	return nil
}

// outlineDest renders the /Dest entry for an outline item, or "" when the
// item has no destination. A malformed or out-of-range PageRef fails the
// document instead of emitting a corrupt /Dest.
func outlineDest(child *Outline, doc *Document) (string, error) {
	if child.PageRef == "" {
		return "", nil
	}

	ref, ok := parseRef(child.PageRef)
	if !ok {
		//nolint:err113 // dynamic values in message
		return "", fmt.Errorf("pdf: outline %q: bad PageRef %q", child.Title, child.PageRef)
	}

	if int(ref) < 1 || int(ref) > len(doc.objects) {
		//nolint:err113 // dynamic values in message
		return "", fmt.Errorf("pdf: outline %q: PageRef %q out of range", child.Title, child.PageRef)
	}

	return fmt.Sprintf("/Dest [%s /XYZ %s %s null]", ref, num(child.X), num(child.Y)), nil
}

func assignRefs(n *Outline, d *Document) {
	n.refStr = d.newObject()
	for _, c := range n.Children {
		assignRefs(c, d)
	}
}

func countChildren(n *Outline) int {
	if len(n.Children) == 0 {
		return 0
	}

	return len(n.Children) + sumChildren(n.Children)
}

func sumChildren(nodes []*Outline) int {
	s := 0
	for _, n := range nodes {
		s += countChildren(n)
	}

	return s
}

func outlineCount(root *Outline) int {
	if len(root.Children) == 0 {
		return 0
	}

	c := 0
	for _, ch := range root.Children {
		c += 1 + countChildren(ch)
	}

	return c
}

// pdfString encodes s as a PDF literal string for a simple WinAnsi/Latin-1
// font. It walks runes (not UTF-8 bytes): each code point ≤ U+00FF becomes
// one string byte so it matches the subset cmap and /Widths indices. Code
// points above U+00FF are folded via winAnsiFold (common punctuation) or
// replaced with '?'; emitting raw UTF-8 bytes made viewers show mojibake
// and missing glyphs (e.g. "·" as "\302\267").
func pdfString(s string) string {
	var buf strings.Builder
	buf.Grow(len(s) + 2)

	buf.WriteByte('(')

	for _, rVal := range s {
		if rVal > maxLatin1Code {
			rVal = winAnsiFold(rVal)
		}

		if rVal > maxLatin1Code {
			rVal = '?'
		}

		cur := byte(rVal)

		switch {
		case cur == '(' || cur == ')' || cur == '\\':
			buf.WriteByte('\\')
			buf.WriteByte(cur)
		case cur < 32 || cur > 126:
			fmt.Fprintf(&buf, "\\%03o", cur)
		default:
			buf.WriteByte(cur)
		}
	}

	buf.WriteByte(')')

	return buf.String()
}

// winAnsiFold maps common Unicode punctuation that appears in HTML/CSS to a
// single-byte Latin-1/WinAnsi stand-in. Unmapped runes are returned unchanged
// (caller substitutes '?').
func winAnsiFold(rVal rune) rune {
	switch rVal {
	case '\u2013', '\u2014': // en/em dash
		return '-'
	case '\u2018', '\u2019': // curly single quotes
		return '\''
	case '\u201C', '\u201D': // curly double quotes
		return '"'
	case '\u2022', '\u2023', '\u25E6', '\u2043': // bullets
		return '\u00B7' // middle dot
	case '\u2026': // ellipsis
		return '.'
	case '\u00A0', '\u2009', '\u200A', '\u2008', '\u2002', '\u2003':
		return ' '
	case '\u00D7', '\u2715', '\u2716': // multiplication / cross marks → ASCII x
		return 'x'
	case '\u00F7': // division sign
		return '/'
	}

	return rVal
}

func pdfDate(t time.Time) string {
	return "D:" + t.UTC().Format("20060102150405") + "Z"
}

func num(v float64) string {
	if v == float64(int(v)) {
		return strconv.Itoa(int(v))
	}

	s := strconv.FormatFloat(v, 'f', 3, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")

	return s
}

type flateState struct {
	buf bytes.Buffer
	zw  *zlib.Writer
}

//nolint:gochecknoglobals // compressor reuse across page streams; not a mutable global
var flatePool sync.Pool

// flateBytes compresses raw with zlib (RFC 1950). PDF /FlateDecode streams
// require the zlib wrapper, not raw DEFLATE (RFC 1951); viewers reject the
// latter and the page appears empty. The compressor is reused across page
// streams; the returned copy owns its bytes before the state goes back to the
// pool.
func flateBytes(raw []byte) []byte {
	state, _ := flatePool.Get().(*flateState)
	if state == nil {
		state = &flateState{} //nolint:exhaustruct // intentional zero-value fields
		state.zw, _ = zlib.NewWriterLevel(&state.buf, zlib.DefaultCompression)
	} else {
		state.buf.Reset()
		state.zw.Reset(&state.buf)
	}

	_, _ = state.zw.Write(raw)
	_ = state.zw.Close()
	out := state.buf.Bytes()
	// Transfer the completed buffer to the caller and give the pooled writer a
	// fresh destination before it is reused. This keeps compressor state pooled
	// without copying every compressed page stream.
	state.buf = bytes.Buffer{}
	state.zw.Reset(&state.buf)
	flatePool.Put(state)

	return out
}

// SortOutlines sorts children by (pageIndex, y, x) - used by layout/outline.
func SortOutlines(nodes []*Outline) {
	sort.SliceStable(nodes, func(i, j int) bool {
		return outlineLess(nodes[i], nodes[j])
	})
}

func outlineLess(left, right *Outline) bool {
	if left.PageRef != right.PageRef {
		return left.PageRef < right.PageRef
	}

	if left.Y != right.Y {
		return left.Y > right.Y // y-down coordinate: higher on page first
	}

	return left.X < right.X
}
