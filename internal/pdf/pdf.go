// Package pdf implements a stdlib-only PDF 1.4 writer: indirect objects,
// xref, catalog/pages tree, content streams (Flate), base-14 fonts, images,
// link annotations, named destinations and outlines. Deterministic output
// for golden tests (creation date is injectable).
package pdf

import (
	"bytes"
	"compress/flate"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Version is the PDF header version emitted.
const Version = "1.4"

// object is one indirect object (pre-serialized dict + optional stream).
type object struct {
	id     int
	dict   string // body text before stream (may be empty)
	stream []byte
}

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
	fontCache      map[string]string // subset key -> font dict ref
	catalogRef     string            // set by finalize
	infoRef        string            // set by finalize
	finalized      bool
}

// NewDocument creates an empty PDF document.
func NewDocument() *Document {
	return &Document{
		info:           map[string]string{},
		useCompression: true,
		fontCache:      map[string]string{},
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

// newObject allocates an indirect object and returns its reference string.
func (d *Document) newObject() string {
	d.nextID++
	id := d.nextID
	d.objects = append(d.objects, &object{id: id})
	return strconv.Itoa(id) + " 0 R"
}

// setDict replaces the object's dict body.
func (d *Document) setDict(ref string, dict string) {
	id := refID(ref)
	d.objects[id-1].dict = dict
}

// setStream attaches a raw stream (compressed later at write time).
func (d *Document) setStream(ref string, raw []byte) {
	id := refID(ref)
	d.objects[id-1].stream = raw
}

func refID(ref string) int {
	end := strings.IndexByte(ref, ' ')
	n, err := strconv.Atoi(ref[:end])
	if err != nil {
		panic("bad ref " + ref)
	}
	return n
}

// Page is one page of the document.
type Page struct {
	doc        *Document
	ref        string
	width      float64
	height     float64
	content    *Content
	contentRef string
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
	annotRef string
}

// AddPage appends a page with the given size in points.
func (d *Document) AddPage(width, height float64) *Page {
	p := &Page{doc: d, width: width, height: height}
	p.ref = d.newObject()
	contentRef := d.newObject()
	p.contentRef = contentRef
	p.content = NewContent()
	p.content.doc = d
	d.pages = append(d.pages, p)
	return p
}

// Width returns the page width in points.
func (p *Page) Width() float64 { return p.width }

// Height returns the page height in points.
func (p *Page) Height() float64 { return p.height }

// Content returns the page content stream builder.
func (p *Page) Content() *Content { return p.content }

// AddLinkURI adds an external URI annotation.
func (p *Page) AddLinkURI(rect [4]float64, uri string) {
	p.annots = append(p.annots, annotation{rect: rect, uri: uri})
}

// AddLinkDest adds an internal GoTo annotation to a page (0-based index).
func (p *Page) AddLinkDest(rect [4]float64, page int, x, y float64) {
	p.annots = append(p.annots, annotation{rect: rect, destPage: page, destX: x, destY: y, hasDest: true})
}

// Outline is a PDF outline (bookmark) node.
type Outline struct {
	Title    string
	PageRef  string // page object ref, set by caller after layout
	X, Y     float64
	Children []*Outline

	refStr string // assigned during finalize
}

// SetOutline installs the document outline tree.
func (d *Document) SetOutline(root *Outline) { d.outlineRoot = root }

// Write serializes the full PDF to w.
func (d *Document) Write(w io.Writer) error {
	if err := d.finalize(); err != nil {
		return err
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%%PDF-%s\n", Version)
	fmt.Fprintln(&buf, "%\xe2\xe3\xcf\xd3") // binary comment

	offsets := make([]int64, len(d.objects)+1)
	for _, o := range d.objects {
		offsets[o.id] = int64(buf.Len())
		if o.dict != "" {
			fmt.Fprintf(&buf, "%d 0 obj\n", o.id)
			buf.WriteString(o.dict)
			if len(o.stream) > 0 {
				buf.WriteString("\nstream\n")
				buf.Write(o.stream)
				buf.WriteString("\nendstream")
			}
			fmt.Fprintln(&buf)
			fmt.Fprintln(&buf, "endobj")
		}
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
	_, err := w.Write(buf.Bytes())
	return err
}

// WriteTo implements io.WriterTo.
func (d *Document) WriteTo(w io.Writer) (int64, error) {
	var buf bytes.Buffer
	if err := d.Write(&buf); err != nil {
		return 0, err
	}
	if _, err := w.Write(buf.Bytes()); err != nil {
		return 0, err
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
		return fmt.Errorf("pdf: no pages")
	}

	catalogRef := d.newObject()
	infoRef := d.newObject()
	pagesRef := d.newObject()

	var pageRefs []string
	for _, p := range d.pages {
		pageRefs = append(pageRefs, p.ref)
	}

	// pages tree root
	d.setDict(pagesRef, fmt.Sprintf(
		"<< /Type /Pages /Kids [%s] /Count %d >>",
		strings.Join(pageRefs, " "), len(pageRefs)))

	catalogParts := []string{
		"<< /Type /Catalog",
		"/Pages " + pagesRef,
	}
	if d.outlineRoot != nil {
		catalogParts = append(catalogParts, "/Outlines "+d.outlineRoot.refStr, "/PageMode /UseOutlines")
	}
	catalogParts = append(catalogParts, ">>")
	d.setDict(catalogRef, strings.Join(catalogParts, " "))

	// info dict
	now := d.creationTime
	if now.IsZero() {
		now = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC) // deterministic default
	}
	infoParts := []string{"<<"}
	for _, k := range []string{"Title", "Subject", "Author", "Keywords"} {
		if v, ok := d.info[k]; ok && v != "" {
			infoParts = append(infoParts, "/"+k+" "+pdfString(v))
		}
	}
	infoParts = append(infoParts,
		"/Creator "+pdfString(d.info["Creator"]),
		"/Producer "+pdfString("gowkhtmltopdf "+Version),
		"/CreationDate "+pdfString(pdfDate(now)),
		"/ModDate "+pdfString(pdfDate(now)),
		">>")
	d.setDict(infoRef, strings.Join(infoParts, " "))

	// per-page objects
	for _, p := range d.pages {
		d.finalizePage(p, pagesRef)
	}

	// outlines
	if d.outlineRoot != nil {
		d.finalizeOutlines(d.outlineRoot)
	}

	d.catalogRef = catalogRef
	d.infoRef = infoRef
	d.finalized = true
	return nil
}

func (d *Document) finalizePage(p *Page, pagesRef string) {
	raw := p.content.Bytes()
	if d.useCompression {
		raw = flateBytes(raw)
		d.setStream(p.contentRef, raw)
		d.setDict(p.contentRef, "<< /Length "+strconv.Itoa(len(raw))+" /Filter /FlateDecode >>")
	} else {
		d.setStream(p.contentRef, raw)
		d.setDict(p.contentRef, "<< /Length "+strconv.Itoa(len(raw))+" >>")
	}

	// resources: fonts + images used by content
	fonts := p.content.fonts()
	imgResources := p.content.imageResources()
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
	if gs := p.content.extGState(); gs != "" {
		res.WriteString(" " + gs)
	}
	res.WriteString(" >>")

	parts := []string{
		"<< /Type /Page",
		"/Parent " + pagesRef,
		fmt.Sprintf("/MediaBox [0 0 %s %s]", num(p.width), num(p.height)),
		"/Resources " + res.String(),
		"/Contents " + p.contentRef,
	}
	if len(p.annots) > 0 {
		d.buildAnnots(p)
		var refs []string
		for _, a := range p.annots {
			refs = append(refs, a.annotRef)
		}
		parts = append(parts, "/Annots ["+strings.Join(refs, " ")+"]")
	}
	parts = append(parts, ">>")
	d.setDict(p.ref, strings.Join(parts, " "))
}

func (d *Document) buildAnnots(p *Page) {
	for i := range p.annots {
		a := &p.annots[i]
		a.annotRef = d.newObject()
		r := a.rect
		var b strings.Builder
		fmt.Fprintf(&b, "<< /Type /Annot /Subtype /Link /Rect [%s %s %s %s]",
			num(r[0]), num(r[1]), num(r[2]), num(r[3]))
		if a.hasDest {
			if a.destPage >= 0 && a.destPage < len(d.pages) {
				fmt.Fprintf(&b, " /Dest [%s /XYZ %s %s null]",
					d.pages[a.destPage].ref, num(a.destX), num(a.destY))
			}
		} else {
			fmt.Fprintf(&b, " /A << /S /URI /URI %s >>", pdfString(a.uri))
		}
		b.WriteString(" >>")
		d.setDict(a.annotRef, b.String())
	}
}

// finalizeOutlines allocates outline item refs and serializes the tree.
// The root Outline is the /Outlines dictionary; its children are items.
func (d *Document) finalizeOutlines(root *Outline) {
	assignRefs(root, d)
	if len(root.Children) == 0 {
		d.setDict(root.refStr, "<< /Type /Outlines /Count 0 >>")
		return
	}
	first := root.Children[0].refStr
	last := root.Children[len(root.Children)-1].refStr
	d.setDict(root.refStr, fmt.Sprintf(
		"<< /Type /Outlines /First %s /Last %s /Count %d >>",
		first, last, outlineCount(root)))
	buildOutlineItems(root, root.refStr, d)
}

// buildOutlineItems writes each child item of parent, with /Parent /Prev
// /Next sibling links, recursing into nested children.
func buildOutlineItems(parent *Outline, parentRef string, d *Document) {
	for i, n := range parent.Children {
		var parts []string
		parts = append(parts, "<< /Title "+pdfString(n.Title))
		if n.PageRef != "" {
			parts = append(parts, fmt.Sprintf("/Dest [%s /XYZ %s %s null]", n.PageRef, num(n.X), num(n.Y)))
		}
		if len(n.Children) > 0 {
			first := n.Children[0].refStr
			last := n.Children[len(n.Children)-1].refStr
			parts = append(parts, fmt.Sprintf("/First %s /Last %s /Count %d", first, last, countChildren(n)))
		}
		parts = append(parts, "/Parent "+parentRef)
		if i > 0 {
			parts = append(parts, "/Prev "+parent.Children[i-1].refStr)
		}
		if i < len(parent.Children)-1 {
			parts = append(parts, "/Next "+parent.Children[i+1].refStr)
		}
		parts = append(parts, ">>")
		d.setDict(n.refStr, strings.Join(parts, " "))
		if len(n.Children) > 0 {
			buildOutlineItems(n, n.refStr, d)
		}
	}
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

func pdfString(s string) string {
	var b strings.Builder
	b.WriteByte('(')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '(' || c == ')' || c == '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		case c < 32 || c > 126:
			fmt.Fprintf(&b, "\\%03o", c)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte(')')
	return b.String()
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

func flateBytes(raw []byte) []byte {
	var buf bytes.Buffer
	fw, _ := flate.NewWriter(&buf, flate.DefaultCompression)
	_, _ = fw.Write(raw)
	_ = fw.Close()
	return buf.Bytes()
}

// SortOutlines sorts children by (pageIndex, y, x) — used by layout/outline.
func SortOutlines(nodes []*Outline) {
	sort.SliceStable(nodes, func(i, j int) bool {
		return outlineLess(nodes[i], nodes[j])
	})
}

func outlineLess(a, b *Outline) bool {
	if a.PageRef != b.PageRef {
		return a.PageRef < b.PageRef
	}
	if a.Y != b.Y {
		return a.Y > b.Y // y-down coordinate: higher on page first
	}
	return a.X < b.X
}
