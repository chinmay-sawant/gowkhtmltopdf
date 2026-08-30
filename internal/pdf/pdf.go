// Package pdf implements a pure-Go version-aware PDF writer (PDF 1.4 default,
// opt-in PDF 1.7 and 2.0, PDF/A-3a/4, and PDF/UA-1/2): indirect objects, xref,
// catalog/pages tree, content streams (Flate), base-14 fonts, images, link
// annotations, named destinations, outlines, trailer /ID, and conformance XMP metadata.
// The write path uses the Go standard library plus one allowlisted exception
// for OpenType shaping (go-text/typesetting). Deterministic output for golden
// tests (creation date is injectable).
package pdf

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"crypto/md5" //nolint:gosec // MD5 is standard for PDF trailer /ID generation
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf16"
)

var (
	errPDFFinalized = errors.New("pdf: document already finalized")
	errPDFNoPages   = errors.New("pdf: no pages")
)

// Version is the default PDF header version emitted for legacy compatibility.
const Version = "1.4"

// object is one indirect object (pre-serialized dict + optional stream).
type object struct {
	id     int
	dict   string // body text before stream (may be empty)
	stream []byte
}

// countingWriter forwards PDF bytes while tracking their exact offset. It
// turns a silent short write into io.ErrShortWrite so xref offsets can never
// be reported as though a truncated stream were complete.
type countingWriter struct {
	w io.Writer
	n int64
}

func (w *countingWriter) Write(data []byte) (int, error) {
	count, err := w.w.Write(data)
	w.n += int64(count)

	if err == nil && count != len(data) {
		err = io.ErrShortWrite
	}

	return count, err
}

func (w *countingWriter) WriteString(text string) (int, error) {
	var (
		count int
		err   error
	)

	if stringWriter, ok := w.w.(io.StringWriter); ok {
		count, err = stringWriter.WriteString(text)
	} else {
		count, err = w.w.Write([]byte(text))
	}

	w.n += int64(count)

	if err == nil && count != len(text) {
		err = io.ErrShortWrite
	}

	return count, err
}

// objRef is a typed indirect-object handle; the "N 0 R" spelling is a
// formatting concern, not a data type, and refs cannot be malformed.
type objRef int

// ObjRef is an alias for objRef for external package access.
type ObjRef = objRef

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

func (d dict) add(k string, v ...string) dict {
	d = append(d, k)

	for _, s := range v {
		d = append(d, s)
	}

	return d
}

func (d dict) String() string { return "<< " + strings.Join(d, " ") + " >>" }

// Document is a PDF under construction.
//
// Document and its Page/Content values have single-goroutine ownership during
// assembly and finalization. Callers must not mutate or write a document
// concurrently; use one document per conversion when parallelism is needed.
type Document struct {
	policy            WriterPolicy
	isUA              bool
	isUA1             bool
	isUA2             bool
	isPDFA3           bool
	isPDFA4           bool
	objects           []*object
	info              map[string]string
	useCompression    bool
	grayscale         bool
	creationTime      time.Time // zero value → deterministic fixed date
	nextID            int
	pages             []*Page
	outlineRoot       *Outline
	fontCache         map[string]objRef // subset key -> font dict ref
	fontRuneSet       map[string]map[rune]struct{}
	fontRunes         map[string][]rune // font resource name -> document-wide rune union (finalize-time)
	fontKeys          map[string]string // font resource name -> precomputed subset cache key
	fontKeyFonts      map[string]*Font  // font resource name -> the face the precomputed key belongs to
	fontFaces         map[string]*Font  // font resource name -> font face (registered during painting)
	fontType0         map[string]bool   // font resource name -> precomputed needsType0(union)
	catalogRef        objRef            // set by finalize
	infoRef           objRef            // set by finalize
	metadataRef       objRef            // set by finalize
	iccRef            objRef            // set by finalize (PDF/A-3, PDF/A-4 sRGB)
	grayIccRef        objRef            // set by finalize (PDF/A-4 Gray)
	outputIntentRef   objRef            // set by finalize (PDF/A-3, PDF/A-4)
	namespaceRef      objRef            // set by finalizeStructure (PDF/UA-2)
	structTreeRootRef objRef            // set by finalizeStructure
	parentTreeRef     objRef            // set by finalizeStructure
	parentTreeNextKey int               // set by finalizeStructure
	structTreeRoot    *StructTreeRoot
	namedDests        []namedDestEntry // dual page+/SD destinations for PDF/UA-2
	lang              string           // document language (default "en-US")
	finalized         bool
}

// namedDestEntry is one PDF 2.0 named destination with a classic page
// destination (/D) and optional structure destination (/SD). Dual form keeps
// Arlington / page-based checkers happy while satisfying PDF/UA-2 clause 8.8.
type namedDestEntry struct {
	name    string
	page    objRef
	x, y    float64
	structR objRef // 0 when no structure destination
}

func (d *Document) updatePolicyFlags() {
	d.isUA1 = d.policy.IsPDFUA1()
	d.isUA2 = d.policy.IsPDFUA2()
	d.isUA = d.isUA1 || d.isUA2
	d.isPDFA3 = d.policy.IsPDFA3()
	d.isPDFA4 = d.policy.IsPDFA4()
}

// IsUA reports whether the document is configured for PDF/UA-1 or PDF/UA-2.
func (d *Document) IsUA() bool { return d.isUA }

// IsUA1 reports whether the document is configured for PDF/UA-1.
func (d *Document) IsUA1() bool { return d.isUA1 }

// IsUA2 reports whether the document is configured for PDF/UA-2.
func (d *Document) IsUA2() bool { return d.isUA2 }

// IsPDFA3 reports whether the document is configured for PDF/A-3.
func (d *Document) IsPDFA3() bool { return d.isPDFA3 }

// IsPDFA4 reports whether the document is configured for PDF/A-4.
func (d *Document) IsPDFA4() bool { return d.isPDFA4 }

func (d *Document) recordFontFace(name string, f *Font) {
	if d.fontFaces == nil {
		d.fontFaces = map[string]*Font{}
	}

	d.fontFaces[name] = f
}

// NewDocument creates an empty PDF document with the default PDF 1.4 policy.
func NewDocument() *Document {
	doc := &Document{ //nolint:exhaustruct // intentional zero-value fields
		policy:         WriterPolicy{Version: PDF14}, //nolint:exhaustruct // default policy
		info:           map[string]string{},
		useCompression: true,
		fontCache:      map[string]objRef{},
		fontRuneSet:    map[string]map[rune]struct{}{},
		fontFaces:      map[string]*Font{},
	}
	doc.updatePolicyFlags()

	return doc
}

// NewDocumentWithPolicy creates an empty PDF document configured with the given writer policy.
func NewDocumentWithPolicy(policy WriterPolicy) (*Document, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}

	doc := NewDocument()
	doc.policy = policy
	doc.updatePolicyFlags()

	return doc, nil
}

// Policy returns the document's writer policy.
func (d *Document) Policy() WriterPolicy {
	return d.policy
}

// Validate checks whether the document's writer policy is valid.
func (d *Document) Validate() error {
	return d.policy.Validate()
}

// SetLang sets the document natural language (e.g. "en-US").
func (d *Document) SetLang(lang string) { d.lang = lang }

// SetLanguage is an alias for SetLang.
func (d *Document) SetLanguage(lang string) { d.lang = lang }

// Lang returns the document natural language.
func (d *Document) Lang() string { return d.lang }

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
	idx := int(r) - 1
	if idx < 0 || idx >= len(d.objects) {
		return
	}

	d.objects[idx].dict = dict
}

// setStream attaches a raw stream (compressed later at write time).
func (d *Document) setStream(r objRef, raw []byte) {
	idx := int(r) - 1
	if idx < 0 || idx >= len(d.objects) {
		return
	}

	d.objects[idx].stream = raw
}

// Page is one page of the document.
type Page struct {
	doc              *Document
	ref              objRef
	width            float64
	height           float64
	content          *Content
	contentRef       objRef
	annots           []annotation
	mcids            []*StructElem
	structParents    int
	hasStructParents bool
}

// annotation is a link annotation.
type annotation struct {
	rect            [4]float64 // x1,y1,x2,y2 in PDF coords
	uri             string     // external link
	destPage        int        // internal link target (0-based page index)
	destX           float64
	destY           float64
	hasDest         bool
	destStruct      *StructElem // optional PDF/UA-2 structure destination target
	annotRef        objRef
	structParent    int
	hasStructParent bool
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
	clonedPage := d.AddPage(src.width, src.height)
	clonedPage.content = cloneContent(src.content)
	clonedPage.annots = append([]annotation(nil), src.annots...)

	for i := range clonedPage.annots {
		clonedPage.annots[i].annotRef = 0
	}

	if len(src.mcids) > 0 {
		clonedPage.mcids = append([]*StructElem(nil), src.mcids...)
		for i, elem := range clonedPage.mcids {
			if elem != nil {
				elem.content = append(elem.content, contentRef{page: clonedPage, mcid: i})
			}
		}
	}

	return clonedPage, nil
}

// Width returns the page width in points.
func (p *Page) Width() float64 { return p.width }

// Height returns the page height in points.
func (p *Page) Height() float64 { return p.height }

// Content returns the page content stream builder.
func (p *Page) Content() *Content { return p.content }

// Doc returns the parent document of this page.
func (p *Page) Doc() *Document { return p.doc }

// AddLinkURI adds an external URI annotation.
func (p *Page) AddLinkURI(rect [4]float64, uri string) ObjRef {
	ref := p.doc.newObject()
	p.annots = append(p.annots, annotation{ //nolint:exhaustruct // intentional zero-value fields
		rect:     rect,
		uri:      uri,
		destPage: 0,
		destX:    0,
		destY:    0,
		hasDest:  false,
		annotRef: ref,
	})

	return ref
}

// AddLinkDest adds an internal GoTo annotation to a page (0-based index).
func (p *Page) AddLinkDest(rect [4]float64, page int, destX, destY float64) ObjRef {
	ref := p.doc.newObject()
	p.annots = append(p.annots, annotation{ //nolint:exhaustruct // intentional zero-value fields
		rect:     rect,
		uri:      "",
		destPage: page,
		destX:    destX,
		destY:    destY,
		hasDest:  true,
		annotRef: ref,
	})

	return ref
}

// SetLinkDestStruct associates a structure element with the most recently
// added internal link annotation on this page. Used for PDF/UA-2 structure
// destinations (ISO 14289-2 clause 8.8). No-op when the page has no annots.
func (p *Page) SetLinkDestStruct(elem *StructElem) {
	if p == nil || elem == nil || len(p.annots) == 0 {
		return
	}

	last := &p.annots[len(p.annots)-1]
	if last.hasDest {
		last.destStruct = elem
	}
}

// Outline is a PDF outline (bookmark) node.
type Outline struct {
	Title      string
	PageRef    string // page object ref, set by caller after layout
	X, Y       float64
	Children   []*Outline
	StructElem *StructElem // optional: associated heading StructElem for PDF/UA-2 /SD

	refStr objRef // assigned during finalize
}

// SetOutline installs the document outline tree.
func (d *Document) SetOutline(root *Outline) { d.outlineRoot = root }

// Write serializes the full PDF to w without staging another complete copy in
// memory. The counting writer supplies the xref offsets as bytes are emitted.
func (d *Document) Write(width io.Writer) error {
	_, err := d.writeTo(width)

	return err
}

// WriteTo implements io.WriterTo.
func (d *Document) WriteTo(width io.Writer) (int64, error) {
	return d.writeTo(width)
}

func writePDFFormat(out *countingWriter, format string, args ...any) error {
	_, err := fmt.Fprintf(out, format, args...)

	if err != nil {
		return fmt.Errorf("write PDF format: %w", err)
	}

	return nil
}

func writePDFString(out *countingWriter, text string) error {
	_, err := io.WriteString(out, text)

	if err != nil {
		return fmt.Errorf("write PDF text: %w", err)
	}

	return nil
}

func writePDFHeader(out *countingWriter, policy WriterPolicy) error {
	if err := writePDFFormat(out, "%%PDF-%s\n", policy.HeaderVersion()); err != nil {
		return err
	}

	return writePDFString(out, "%\xe2\xe3\xcf\xd3\n") // binary comment
}

func writePDFObject(out *countingWriter, obj *object) (int64, error) {
	offset := out.n

	if err := writePDFFormat(out, "%d 0 obj\n", obj.id); err != nil {
		return 0, err
	}

	if err := writePDFString(out, obj.dict); err != nil {
		return 0, err
	}

	if len(obj.stream) > 0 {
		if err := writePDFString(out, "\nstream\n"); err != nil {
			return 0, err
		}

		if _, err := out.Write(obj.stream); err != nil {
			return 0, err
		}

		if err := writePDFString(out, "\nendstream"); err != nil {
			return 0, err
		}
	}

	if err := writePDFString(out, "\nendobj\n"); err != nil {
		return 0, err
	}

	return offset, nil
}

func computeTrailerID(doc *Document) string {
	hasher := md5.New() //nolint:gosec // MD5 is standard for PDF trailer /ID generation

	_, _ = hasher.Write([]byte(doc.policy.HeaderVersion()))

	now := doc.creationTime
	if now.IsZero() {
		now = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	_, _ = hasher.Write([]byte(pdfDate(now)))

	for _, k := range sortedStringKeys(doc.info) {
		_, _ = hasher.Write([]byte(k))
		_, _ = hasher.Write([]byte(doc.info[k]))
	}

	_, _ = fmt.Fprintf(hasher, "pages:%d", len(doc.pages))

	for _, p := range doc.pages {
		_, _ = fmt.Fprintf(hasher, "page:%f,%f", p.width, p.height)
	}

	for _, obj := range doc.objects {
		_, _ = fmt.Fprintf(hasher, "obj:%d:%s:%d", obj.id, obj.dict, len(obj.stream))
	}

	sum := hasher.Sum(nil)

	return fmt.Sprintf("%032X", sum)
}

//nolint:cyclop // xref subsection + trailer shape branches by version/profile
func writePDFTrailer(out *countingWriter, doc *Document, offsets []int64) error {
	xrefPos := out.n

	if err := writePDFFormat(out, "xref\n0 %d\n", len(doc.objects)+1); err != nil {
		return err
	}

	if err := writePDFString(out, "0000000000 65535 f \n"); err != nil {
		return err
	}

	for idx := 1; idx <= len(doc.objects); idx++ {
		if err := writePDFFormat(out, "%010d 00000 n \n", offsets[idx]); err != nil {
			return err
		}
	}

	if err := writePDFString(out, "trailer\n"); err != nil {
		return err
	}

	switch {
	case doc.policy.IsPDFA4():
		idHex := computeTrailerID(doc)
		if err := writePDFFormat(
			out, "<< /Size %d /Root %s /ID [ <%s> <%s> ] >>\n",
			len(doc.objects)+1, doc.catalogRef, idHex, idHex,
		); err != nil {
			return err
		}
	case doc.policy.Version >= PDF17:
		idHex := computeTrailerID(doc)
		if err := writePDFFormat(
			out, "<< /Size %d /Root %s /Info %s /ID [ <%s> <%s> ] >>\n",
			len(doc.objects)+1, doc.catalogRef, doc.infoRef, idHex, idHex,
		); err != nil {
			return err
		}
	default:
		if err := writePDFFormat(
			out, "<< /Size %d /Root %s /Info %s >>\n", len(doc.objects)+1, doc.catalogRef, doc.infoRef,
		); err != nil {
			return err
		}
	}

	return writePDFFormat(out, "startxref\n%d\n%%%%EOF\n", xrefPos)
}

const pdfBufferSize = 64 * 1024

func (d *Document) writeTo(width io.Writer) (int64, error) {
	if err := d.finalize(); err != nil {
		return 0, err
	}

	bufWriter := bufio.NewWriterSize(width, pdfBufferSize)
	out := &countingWriter{w: bufWriter} //nolint:exhaustruct // count starts at zero

	if err := writePDFHeader(out, d.policy); err != nil {
		return out.n, fmt.Errorf("pdf: write: %w", err)
	}

	offsets := make([]int64, len(d.objects)+1)

	for _, obj := range d.objects {
		if obj.dict == "" {
			// Object was allocated but never materialized; leave its offset
			// unrecorded so the xref cannot point at the *next* object.
			continue
		}

		offset, err := writePDFObject(out, obj)

		if err != nil {
			return out.n, fmt.Errorf("pdf: write: %w", err)
		}

		offsets[obj.id] = offset
	}

	if err := writePDFTrailer(out, d, offsets); err != nil {
		return out.n, fmt.Errorf("pdf: write: %w", err)
	}

	if err := bufWriter.Flush(); err != nil {
		return out.n, fmt.Errorf("pdf: flush: %w", err)
	}

	return out.n, nil
}

// embedICC embeds an ICC profile stream object with compression.
func (d *Document) embedICC(n int, alt string, rawFlated []byte) objRef {
	ref := d.newObject()
	d.setDict(ref, fmt.Sprintf("<< /N %d /Alternate /%s /Filter /FlateDecode /Length %d >>", n, alt, len(rawFlated)))
	d.setStream(ref, rawFlated)

	return ref
}

// embedOutputIntents creates output intent dictionary and embeds required ICC profiles.
func (d *Document) embedOutputIntents() {
	const (
		rgbChannels  = 3
		grayChannels = 1
	)

	if d.isPDFA3 {
		d.iccRef = d.embedICC(rgbChannels, "DeviceRGB", FlatedSRGBICCProfile())
		d.outputIntentRef = d.newObject()
		d.setDict(d.outputIntentRef, outputIntentDict(d.iccRef))
	} else if d.isPDFA4 {
		d.iccRef = d.embedICC(rgbChannels, "DeviceRGB", FlatedSRGBICCProfile())
		d.grayIccRef = d.embedICC(grayChannels, "DeviceGray", FlatedGrayICCProfile())
		d.outputIntentRef = d.newObject()
		d.setDict(d.outputIntentRef, outputIntentDict(d.iccRef))
	}
}

// embedMetadata creates the document's XMP metadata object when Version >= PDF 1.7.
func (d *Document) embedMetadata() objRef {
	if d.policy.Version < PDF17 {
		return 0
	}

	metadataRef := d.newObject()
	xmpBytes := d.buildXMPMetadata()
	d.setDict(metadataRef, fmt.Sprintf("<< /Type /Metadata /Subtype /XML /Length %d >>", len(xmpBytes)))
	d.setStream(metadataRef, xmpBytes)

	return metadataRef
}

// finalize builds catalog, pages tree, fonts, images, annots, outlines and
// page objects once.
//
//nolint:cyclop,funlen // finalize coordinates entire document serialization pipeline
func (d *Document) finalize() error {
	if d.finalized {
		return nil
	}

	if err := d.policy.Validate(); err != nil {
		return err
	}

	if len(d.pages) == 0 {
		return errPDFNoPages
	}

	if d.isUA {
		title := strings.TrimSpace(d.info["Title"])
		if title == "" {
			return ErrTitleRequired
		}
	}

	catalogRef := d.newObject()

	var infoRef objRef
	if !d.isPDFA4 {
		infoRef = d.newObject()
	}

	pagesRef := d.newObject()
	metadataRef := d.embedMetadata()
	d.embedOutputIntents()

	d.unionFontRunes()

	pageRefs := make([]string, 0, len(d.pages))
	for _, p := range d.pages {
		pageRefs = append(pageRefs, p.ref.String())
	}

	// pages tree root
	d.setDict(pagesRef, fmt.Sprintf(
		"<< /Type /Pages\n/Kids [%s]\n/Count %d >>",
		strings.Join(pageRefs, " "), len(pageRefs)))

	// Structure tree must be finalized before outlines/annots so StructElem
	// refs are available for PDF/UA-2 structure destinations (/SD).
	if err := d.finalizeStructure(); err != nil {
		return err
	}

	// Outlines and page annots register dual named destinations under UA-2
	// before the catalog is written (catalog needs /Names /Dests).
	if d.outlineRoot != nil {
		if err := d.finalizeOutlines(d.outlineRoot); err != nil {
			return err
		}
	}

	for _, p := range d.pages {
		if err := d.finalizePage(p, pagesRef); err != nil {
			return err
		}
	}

	namesRef := d.serializeNamedDests()
	d.setDict(catalogRef, d.catalogDict(pagesRef, metadataRef, namesRef))

	if infoRef != 0 {
		d.setDict(infoRef, d.infoDict())
	}

	d.catalogRef = catalogRef
	d.infoRef = infoRef
	d.metadataRef = metadataRef
	d.finalized = true

	return nil
}

// unionFontRunes materializes the document-wide rune sets collected while
// content streams were painted. Keeping one set on Document avoids retaining
// a duplicate rune slice on every page until finalization.
func (d *Document) unionFontRunes() {
	if len(d.fontRuneSet) == 0 {
		return
	}

	d.fontRunes = make(map[string][]rune, len(d.fontRuneSet))
	d.fontKeys = make(map[string]string, len(d.fontRuneSet))
	d.fontKeyFonts = make(map[string]*Font, len(d.fontRuneSet))
	d.fontType0 = make(map[string]bool, len(d.fontRuneSet))

	for _, name := range sortedStringKeys(d.fontRuneSet) {
		used := d.fontRuneSet[name]
		runes := make([]rune, 0, len(used))

		for rVal := range used {
			runes = append(runes, rVal)
		}

		sort.Slice(runes, func(i, j int) bool { return runes[i] < runes[j] })
		d.fontRunes[name] = runes

		fnt := d.fontFaces[name]
		if fnt == nil {
			for _, page := range d.pages {
				if fnt = page.content.fontFiles[name]; fnt != nil {
					break
				}
			}
		}

		if fnt == nil {
			continue
		}

		type0 := needsType0(runes)
		d.fontType0[name] = type0

		mode := 0
		if type0 {
			mode = 1
		}

		baseName := fnt.PostScriptName
		if baseName == "" {
			baseName = fallbackFontName
		}

		d.fontKeyFonts[name] = fnt
		d.fontKeys[name] = fmt.Sprintf("v%d|%x|%s|%s", mode, fnt.fingerprint, baseName, runesKey(runes))
	}
}

func sortedStringKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}

// catalogDict builds the /Catalog dictionary, wiring /Metadata when present,
// /Outlines when set, /OutputIntents when present, and PDF/UA-1 & PDF/UA-2 keys when enabled.
//
// Catalog /Version is deliberately not emitted: the file header is the single
// version authority. This matches the PDF 1.7 sibling decision (#31) and keeps
// the header and catalog from ever disagreeing. ISO 32000-2 keeps the header
// authoritative; catalog /Version was deprecated in 1.4 and is optional there.
func (d *Document) catalogDict(pagesRef, metadataRef, namesRef objRef) string {
	cat := dict{}.add("/Type", "/Catalog")
	if metadataRef != 0 {
		cat = cat.add("/Metadata", metadataRef.String())
	}

	cat = cat.add("/Pages", pagesRef.String())
	if d.outlineRoot != nil {
		cat = cat.add("/Outlines", d.outlineRoot.refStr.String()).add("/PageMode", "/UseOutlines")
	}

	if namesRef != 0 {
		cat = cat.add("/Names", namesRef.String())
	}

	if d.outputIntentRef != 0 {
		cat = cat.add("/OutputIntents", "["+d.outputIntentRef.String()+"]")
	}

	if d.policy.IsPDFUA1() || d.policy.IsPDFUA2() {
		cat = cat.add("/MarkInfo", "<< /Marked true >>")

		lang := d.lang
		if lang == "" {
			lang = "en-US"
		}

		cat = cat.add("/Lang", pdfString(lang))
		cat = cat.add("/ViewerPreferences", "<< /DisplayDocTitle true >>")

		if d.structTreeRootRef != 0 {
			cat = cat.add("/StructTreeRoot", d.structTreeRootRef.String())
		}
	}

	return cat.String()
}

// registerDualDest records a PDF 2.0 named destination with page /D and
// optional structure /SD, returning the destination name for /Dest (name).
func (d *Document) registerDualDest(page objRef, left, top float64, structElem *StructElem) string {
	name := fmt.Sprintf("D%d", len(d.namedDests)+1)
	entry := namedDestEntry{ //nolint:exhaustruct // intentional zero-value fields
		name: name,
		page: page,
		x:    left,
		y:    top,
	}

	if structElem != nil && structElem.ref != 0 {
		entry.structR = structElem.ref
	}

	d.namedDests = append(d.namedDests, entry)

	return name
}

// serializeNamedDests writes Catalog /Names /Dests as a name tree of dual
// destinations. Returns 0 when no named destinations were registered.
func (d *Document) serializeNamedDests() objRef {
	if len(d.namedDests) == 0 {
		return 0
	}

	const nameAndRefPair = 2

	nameParts := make([]string, 0, len(d.namedDests)*nameAndRefPair)

	for _, entry := range d.namedDests {
		destObj := d.newObject()
		destDict := fmt.Sprintf("<< /D [%s /XYZ %s %s null]",
			entry.page, num(entry.x), num(entry.y))

		if entry.structR != 0 {
			destDict += fmt.Sprintf(" /SD [%s /XYZ %s %s null]",
				entry.structR, num(entry.x), num(entry.y))
		}

		destDict += " >>"
		d.setDict(destObj, destDict)
		nameParts = append(nameParts, pdfString(entry.name), destObj.String())
	}

	treeRef := d.newObject()
	d.setDict(treeRef, fmt.Sprintf("<< /Names [ %s ] >>", strings.Join(nameParts, " ")))

	namesRef := d.newObject()
	d.setDict(namesRef, fmt.Sprintf("<< /Dests %s >>", treeRef.String()))

	return namesRef
}

// infoDict builds the /Info dictionary with the (injectable) timestamps.
func (d *Document) infoDict() string {
	now := d.creationTime
	if now.IsZero() {
		now = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC) // deterministic default
	}

	info := dict{}
	for _, k := range []string{"Title", "Subject", "Author", "Keywords", "Creator"} {
		if v, ok := d.info[k]; ok && v != "" {
			info = info.add("/"+k, d.encodeTextString(v))
		}
	}

	return info.add("/Producer", d.encodeTextString(d.policy.ProducerVersion())).
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

	res, err := buildPageResources(page.content, d.iccRef, d.grayIccRef, d.policy.Version)
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

	if (d.policy.IsPDFUA1() || d.policy.IsPDFUA2()) && page.hasStructParents {
		parts = append(parts, fmt.Sprintf("/StructParents %d", page.structParents))
	}

	if len(page.annots) > 0 {
		d.buildAnnots(page)

		var refs []string
		for _, a := range page.annots {
			refs = append(refs, a.annotRef.String())
		}

		parts = append(parts, "/Annots ["+strings.Join(refs, " ")+"]")
		if d.policy.IsPDFUA1() || d.policy.IsPDFUA2() {
			parts = append(parts, "/Tabs /S")
		}
	}

	parts = append(parts, ">>")
	d.setDict(page.ref, strings.Join(parts, "\n"))

	return nil
}

// buildPageResources assembles the /Resources dict from the content's fonts,
// images, and color spaces. PDF 2.0 pages omit /ProcSet (removed in
// ISO 32000-2; already obsolete in ISO 32000-1 §14.2), so the dict lists
// only the resources the content actually uses.
//
//nolint:wsl,cyclop,funlen // resource dictionary assembly is a linear PDF serialization block
func buildPageResources(content *Content, iccRef, grayIccRef objRef, version PDFVersion) (string, error) {
	fonts, err := content.fonts()
	if err != nil {
		return "", err
	}

	imgResources := content.imageResources()

	var res strings.Builder

	if version < PDF20 {
		res.WriteString("<< /ProcSet [/PDF /Text /ImageB /ImageC /ImageI]")
	} else {
		res.WriteString("<<")
	}

	if iccRef != 0 || grayIccRef != 0 {
		res.WriteString(" /ColorSpace <<")
		if iccRef != 0 {
			res.WriteString(" /DefaultRGB [/ICCBased " + iccRef.String() + "]")
		}
		if grayIccRef != 0 {
			res.WriteString(" /DefaultGray [/ICCBased " + grayIccRef.String() + "]")
		}
		res.WriteString(" >>")
	}

	if len(fonts) > 0 {
		res.WriteString(" /Font <<")

		for _, name := range sortedStringKeys(fonts) {
			ref := fonts[name]
			res.WriteString(" /")
			res.WriteString(name)
			res.WriteString(" ")
			res.WriteString(ref)
		}

		res.WriteString(" >>")
	}

	if len(imgResources) > 0 {
		res.WriteString(" /XObject <<")

		for _, name := range sortedStringKeys(imgResources) {
			ref := imgResources[name]
			res.WriteString(" /")
			res.WriteString(name)
			res.WriteString(" ")
			res.WriteString(ref)
		}

		res.WriteString(" >>")
	}

	if gs := content.extGState(); gs != "" {
		res.WriteString(" ")
		res.WriteString(gs)
	}

	res.WriteString(" >>")

	return res.String(), nil
}

func annotDescription(arg *annotation) string {
	if arg.uri != "" {
		return arg.uri
	}

	if arg.hasDest {
		return fmt.Sprintf("Link to page %d", arg.destPage+1)
	}

	return "Link"
}

// structureDestElem picks the StructElem for a PDF/UA-2 /SD entry. Prefer an
// explicit per-annot target; otherwise the first marked-content owner on the
// destination page (has /Pg association). Never fall back to Document alone —
// a structure destination without a page is an invalid page destination.
func structureDestElem(arg *annotation, doc *Document) *StructElem {
	if arg == nil {
		return nil
	}

	if arg.destStruct != nil && arg.destStruct.ref != 0 {
		return arg.destStruct
	}

	if doc == nil || !arg.hasDest || arg.destPage < 0 || arg.destPage >= len(doc.pages) {
		return nil
	}

	return firstPageStructElem(doc.pages[arg.destPage])
}

func firstPageStructElem(page *Page) *StructElem {
	if page == nil {
		return nil
	}

	for _, owner := range page.mcids {
		if owner != nil && owner.ref != 0 {
			return owner
		}
	}

	return nil
}

func writeAnnotDest(buf *strings.Builder, doc *Document, arg *annotation) {
	if arg.destPage < 0 || arg.destPage >= len(doc.pages) {
		return
	}

	pageRef := doc.pages[arg.destPage].ref
	// PDF/UA-2: dual named dest — /D page (Arlington/PDF/A) + /SD struct (UA-2 8.8).
	if doc.policy.IsPDFUA2() {
		name := doc.registerDualDest(pageRef, arg.destX, arg.destY, structureDestElem(arg, doc))
		fmt.Fprintf(buf, " /Dest %s", pdfString(name))

		return
	}

	fmt.Fprintf(buf, " /Dest [%s /XYZ %s %s null]",
		pageRef, num(arg.destX), num(arg.destY))
}

func (d *Document) buildAnnots(page *Page) {
	for i := range page.annots {
		arg := &page.annots[i]
		if arg.annotRef == 0 {
			arg.annotRef = d.newObject()
		}

		r := arg.rect

		var buf strings.Builder

		fmt.Fprintf(&buf, "<< /Type /Annot /Subtype /Link /Rect [%s %s %s %s] /Border [0 0 0] /F 4",
			num(r[0]), num(r[1]), num(r[2]), num(r[3]))

		if d.policy.IsPDFUA1() || d.policy.IsPDFUA2() {
			fmt.Fprintf(&buf, " /Contents %s", d.encodeTextString(annotDescription(arg)))

			if arg.hasStructParent {
				fmt.Fprintf(&buf, " /StructParent %d", arg.structParent)
			}
		}

		if arg.hasDest {
			writeAnnotDest(&buf, d, arg)
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
		parts = append(parts, "<< /Title "+doc.encodeTextString(child.Title))

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
//
// Under PDF/UA-2, destinations are dual named destinations: /D is a page
// XYZ dest (Arlington DestXYZ + PDF/A page validity) and /SD is a structure
// destination (ISO 14289-2:2024 clause 8.8). The outline item may also carry
// /SE pointing at the heading StructElem.
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

	if doc.policy.IsPDFUA2() {
		name := doc.registerDualDest(ref, child.X, child.Y, child.StructElem)
		parts := "/Dest " + pdfString(name)

		if child.StructElem != nil && child.StructElem.ref != 0 {
			parts += " /SE " + child.StructElem.ref.String()
		}

		return parts, nil
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
	const literalDelims = 2 // '(' and ')' framing the literal

	return string(appendPDFString(make([]byte, 0, len(s)+literalDelims), s))
}

// winAnsiFold maps common Unicode punctuation that appears in HTML/CSS to a
// simple-font stand-in. Dashes outside Latin-1 remain unchanged so visible
// text uses the Type0 path and retains the actual em/en dash glyph.
func winAnsiFold(rVal rune) rune {
	switch rVal {
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

// pdfDocEncodingFold maps punctuation to PDF document-string bytes. Unlike
// visible page text, metadata and outline strings do not have a font subset,
// so their standard PDFDocEncoding dash bytes are the lossless representation.
func pdfDocEncodingFold(rVal rune) rune {
	const (
		pdfDocEnDash = 0x96
		pdfDocEmDash = 0x97
	)

	switch rVal {
	case '\u2013':
		return pdfDocEnDash
	case '\u2014':
		return pdfDocEmDash
	}

	return winAnsiFold(rVal)
}

func pdfDate(t time.Time) string {
	return "D:" + t.UTC().Format("20060102150405") + "Z"
}

// encodeTextString encodes text as a PDF text string according to document version.
// For PDF 1.4, it uses pdfString (Latin-1 / PDFDocEncoding fold).
// For PDF 1.7, if text contains any characters outside Latin-1 (> U+00FF),
// it encodes text as UTF-16BE with a byte-order mark (BOM \xFE\xFF) formatted as a hex string <FEFF...>.
// For PDF 2.0 (ISO 32000-2), if text contains any characters outside Latin-1,
// it encodes text as a UTF-8 text string (BOM EF BB BF + UTF-8) as a hex string.
func encodeTextString(text string, version PDFVersion) string {
	if text == "" {
		return "()"
	}

	if version < PDF17 {
		return pdfString(text)
	}

	for _, rVal := range text {
		if rVal > maxLatin1Code {
			if version >= PDF20 {
				return encodeUTF8Hex(text)
			}

			return encodeUTF16BEHex(text)
		}
	}

	return pdfString(text)
}

const hexUpperDigits = "0123456789ABCDEF"

// encodeUTF8Hex renders text as a PDF 2.0 UTF-8 text string: ISO 32000-2
// requires UTF-8 text strings to begin with U+FEFF (bytes EF BB BF), so the
// hex string is the BOM followed by the UTF-8 encoding of text. Hex form
// avoids literal-string escaping for bytes outside printable ASCII.
func encodeUTF8Hex(text string) string {
	utf8Bytes := []byte(text)

	var buf strings.Builder

	buf.Grow(len(utf8Bytes)*2 + len("<EFBBBF>"))
	buf.WriteString("<EFBBBF")

	for _, b := range utf8Bytes {
		buf.WriteByte(hexUpperDigits[b>>4])
		buf.WriteByte(hexUpperDigits[b&0x0F])
	}

	buf.WriteByte('>')

	return buf.String()
}

//nolint:mnd // 4-bit nibble shift offsets for 16-bit hex conversion
func encodeUTF16BEHex(text string) string {
	const (
		bomLen       = 2
		hexPerUnit   = 4
		prefixBOMHex = "<FEFF"
	)

	u16 := utf16.Encode([]rune(text))

	var buf strings.Builder

	buf.Grow(bomLen + len(prefixBOMHex) + len(u16)*hexPerUnit)
	buf.WriteString(prefixBOMHex)

	for _, unit := range u16 {
		buf.WriteByte(hexUpperDigits[(unit>>12)&0x0F])
		buf.WriteByte(hexUpperDigits[(unit>>8)&0x0F])
		buf.WriteByte(hexUpperDigits[(unit>>4)&0x0F])
		buf.WriteByte(hexUpperDigits[unit&0x0F])
	}

	buf.WriteByte('>')

	return buf.String()
}

func (d *Document) encodeTextString(text string) string {
	return encodeTextString(text, d.policy.Version)
}

func num(v float64) string {
	var buf [24]byte

	return string(appendPDFNum(buf[:0], v))
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
const maxPooledFlateBufferSize = 16 * 1024 * 1024 // 16 MiB max retention
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

	res := append([]byte(nil), state.buf.Bytes()...)
	stateBufCap := state.buf.Cap()

	if stateBufCap <= maxPooledFlateBufferSize {
		flatePool.Put(state)
	}

	return res
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
