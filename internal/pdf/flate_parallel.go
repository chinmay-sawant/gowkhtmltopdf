package pdf

import (
	"compress/zlib"
	"runtime"
	"sync"
)

// maxPageFlateWorkers caps the retained page-stream worker set. The writer A/B
// probe for this phase (plans/0.2.6/perf-time/profiles/pagination-paint-deep.md
// section 5.2) measured a full removal of the flate compute wall at 500 pages
// (median -150.5 ms) with eight workers; more workers add retained zlib state
// (~663 KB per worker) without adding throughput on this workload.
const maxPageFlateWorkers = 8

// flateJob is one page stream handed to a retained worker. out points at the
// job's own slot in the caller's output slice, so workers never write the same
// slot; waitGroup is the caller's batch barrier.
type flateJob struct {
	raw       []byte
	out       *[]byte
	waitGroup *sync.WaitGroup
}

// pageFlatePool compresses page streams on a worker set retained for the
// process lifetime. Each worker owns one flateState, so the ~663 KB zlib
// writer is allocated once, not per conversion: a per-call pool (or a
// sync.Pool of writers, which a GC between conversions empties) measured
// +6.0 MB B/op and is rejected. Workers park on jobs between documents and
// are never created or stopped per document, so no goroutine is tied to a
// document's lifetime.
type pageFlatePool struct {
	once sync.Once
	jobs chan flateJob
}

//nolint:gochecknoglobals // process-lifetime retained workers; see pageFlatePool
var retainedPageFlate pageFlatePool

// start launches the worker set once. GOMAXPROCS is read at first use, not at
// package init, so a test or caller that lowers it is respected.
func (p *pageFlatePool) start() {
	workers := min(runtime.GOMAXPROCS(0), maxPageFlateWorkers)

	p.jobs = make(chan flateJob)

	for range workers {
		go p.worker()
	}
}

// worker owns one compressor state for its lifetime.
func (p *pageFlatePool) worker() {
	state := &flateState{} //nolint:exhaustruct // intentional zero-value fields
	state.zw, _ = zlib.NewWriterLevel(&state.buf, zlib.DefaultCompression)

	for job := range p.jobs {
		*job.out = state.compress(job.raw)
		job.waitGroup.Done()
	}
}

// compress flates every raw stream and returns the outputs in input order.
// Output order never depends on which worker finishes first, so serialization
// stays byte-deterministic.
func (p *pageFlatePool) compress(raws [][]byte) [][]byte {
	p.once.Do(p.start)

	outputs := make([][]byte, len(raws))

	var waitGroup sync.WaitGroup

	waitGroup.Add(len(raws))

	for index := range raws {
		p.jobs <- flateJob{raw: raws[index], out: &outputs[index], waitGroup: &waitGroup}
	}

	waitGroup.Wait()

	return outputs
}

// finalizePages materializes every page stream and page object. With
// compression on, pages are flated in windows of at most maxPageFlateWorkers
// then each window is finalized in input-index order before the next window
// starts. Peak live compressed copies equal the window, not the page count.
// The resource pass stays serial because it mutates the shared object table
// and font cache.
func (d *Document) finalizePages(pagesRef objRef) error {
	if !d.useCompression {
		for _, page := range d.pages {
			if err := d.finalizePage(page, pagesRef, page.content.Bytes()); err != nil {
				return err
			}
		}

		return nil
	}

	if !flatePagesInParallel(len(d.pages)) {
		return d.finalizePagesSerialFlate(pagesRef)
	}

	return d.finalizePagesWindowed(pagesRef)
}

// flatePagesInParallel reports whether a document with pageCount page streams
// should use the retained worker set. A single-page document never starts the
// workers, and a single-CPU run has nothing to overlap.
func flatePagesInParallel(pageCount int) bool {
	return pageCount > 1 && runtime.GOMAXPROCS(0) > 1
}

// finalizePagesSerialFlate flates then finalizes one page at a time so a
// single-page document (or a single-CPU run) never starts the retained
// workers and never holds a second compressed copy.
func (d *Document) finalizePagesSerialFlate(pagesRef objRef) error {
	for _, page := range d.pages {
		stream := flateBytes(page.content.Bytes())
		if err := d.finalizePage(page, pagesRef, stream); err != nil {
			return err
		}
	}

	return nil
}

// finalizePagesWindowed flates at most maxPageFlateWorkers pages on the
// retained worker set, finalizes that window in index order, then compresses
// the next window. Output order is the input index, never worker completion
// order. finalizePage still drops each raw buffer (PDF-06).
func (d *Document) finalizePagesWindowed(pagesRef objRef) error {
	for start := 0; start < len(d.pages); start += maxPageFlateWorkers {
		end := min(start+maxPageFlateWorkers, len(d.pages))
		if err := d.finalizeCompressedWindow(pagesRef, d.pages[start:end]); err != nil {
			return err
		}
	}

	return nil
}

// finalizeCompressedWindow flates one window through the retained workers
// then finalizes in slice order. The window slice is the only extra live
// compressed copies; each slot is cleared after finalizePage so the extra
// reference dies before the next page is retained.
func (d *Document) finalizeCompressedWindow(pagesRef objRef, pages []*Page) error {
	raws := make([][]byte, len(pages))
	for index, page := range pages {
		raws[index] = page.content.Bytes()
	}

	streams := retainedPageFlate.compress(raws)

	for index, page := range pages {
		if err := d.finalizePage(page, pagesRef, streams[index]); err != nil {
			return err
		}

		// finalizePage already dropped the builder's ref (PDF-06). Clear the
		// window aliases so neither the raw backing array nor the extra
		// compressed pointer stays live while later slots in this window
		// finalize.
		raws[index] = nil
		streams[index] = nil
	}

	return nil
}
