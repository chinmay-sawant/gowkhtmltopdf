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
// compression on, all page streams are flated first (in parallel through the
// retained worker set); the resource pass stays serial because it mutates the
// shared object table and font cache.
func (d *Document) finalizePages(pagesRef objRef) error {
	if !d.useCompression {
		for _, page := range d.pages {
			if err := d.finalizePage(page, pagesRef, page.content.Bytes()); err != nil {
				return err
			}
		}

		return nil
	}

	streams := d.compressPageStreams()

	for index, page := range d.pages {
		if err := d.finalizePage(page, pagesRef, streams[index]); err != nil {
			return err
		}
	}

	return nil
}

// flatePagesInParallel reports whether a document with pageCount page streams
// should use the retained worker set. A single-page document never starts the
// workers, and a single-CPU run has nothing to overlap.
func flatePagesInParallel(pageCount int) bool {
	return pageCount > 1 && runtime.GOMAXPROCS(0) > 1
}

// compressPageStreams flates every page content buffer and returns the
// compressed streams in page order. Single-page documents (and single-CPU
// runs) take the serial flateBytes path, so the worker set is never started
// for them. Raw buffers are only borrowed while compression runs; the
// returned slice holds no reference to them, so each raw buffer becomes
// collectable as soon as its page releases it (PDF-06).
func (d *Document) compressPageStreams() [][]byte {
	raws := make([][]byte, len(d.pages))
	for index, page := range d.pages {
		raws[index] = page.content.Bytes()
	}

	if !flatePagesInParallel(len(raws)) {
		streams := make([][]byte, len(raws))
		for index := range raws {
			streams[index] = flateBytes(raws[index])
		}

		return streams
	}

	return retainedPageFlate.compress(raws)
}
