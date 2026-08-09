package convert

import (
	"context"
	"fmt"

	"gowkhtmltopdf/internal/outline"
)

// pdfPipeline adapts the PDF-specific private state to render.Pipeline. The
// render package controls lifecycle ordering; this adapter owns PDF details
// such as TOC assembly, page copies, links, and header/footer placement.
type pdfPipeline struct {
	run *runContext
}

func (p *pdfPipeline) RenderObjects(ctx context.Context) error {
	tocs, bodies, err := p.run.renderObjects(ctx)
	if err != nil {
		return err
	}

	p.run.tocs = tocs
	p.run.bodies = bodies
	p.run.headings = flatHeadings(bodies)

	return nil
}

func (p *pdfPipeline) Assemble(ctx context.Context) error {
	p.run.exclude = parseExcludeSelectors(p.run.req.Global.ExcludeFromOutline, p.run.log)

	if err := p.assembleTOC(ctx); err != nil {
		return err
	}

	if err := p.assembleOutline(); err != nil {
		return err
	}

	p.assembleLinks()

	if err := p.assembleDocument(); err != nil {
		return err
	}

	if err := p.assembleCopies(); err != nil {
		return err
	}

	return p.assembleHeadersFooters(ctx)
}

func (p *pdfPipeline) assembleTOC(ctx context.Context) error {
	run := p.run
	if len(run.tocs) == 0 {
		return nil
	}

	run.report("Building table of contents", percent(len(run.req.Objects), len(run.req.Objects)+1))
	tocTree := outline.BuildTreeBy(run.headings, outline.Options{ //nolint:exhaustruct // intentional zero-value fields
		Exclude: run.exclude,
	}, outline.DocumentPage)

	tocTotal, err := renderTOCObjects(ctx, run.font, run.doc, run.req, run.tocs, tocTree.Flatten(), run.log)
	if err != nil {
		return err
	}

	run.tocTotal = tocTotal
	order := tocFirstOrder(run.tocs, run.bodies)

	if err := run.doc.ReorderPages(order); err != nil {
		return fmt.Errorf("toc assembly: %w", err)
	}

	pos := 0
	for _, tr := range run.tocs {
		tr.start = pos
		pos += tr.tocPages
	}

	for _, bg := range run.bodies {
		bg.start = run.tocTotal + bg.offset
	}

	return nil
}

func (p *pdfPipeline) assembleOutline() error {
	run := p.run
	if !run.req.Global.Outline {
		return nil
	}

	outTree := outline.BuildTreeBy(run.headings, outline.Options{
		MaxDepth: run.req.Global.OutlineDepth,
		Exclude:  run.exclude,
	}, outline.DocumentPage)
	if run.req.Global.DumpOutline {
		xml := outline.DumpOutlineXMLBy(outTree, run.tocTotal, outline.DocumentPage)
		if _, err := run.req.OutlineOutput.Write(xml); err != nil {
			return fmt.Errorf("dump outline: %w", err)
		}
	}

	root := emitOutline(run.doc, outTree, run.bodies, run.tocTotal)
	if len(root.Children) > 0 {
		run.doc.SetOutline(root)
	}

	return nil
}

func (p *pdfPipeline) assembleLinks() {
	run := p.run
	if len(run.tocs) > 0 {
		applyTOCLinks(run.doc, run.tocs, run.bodies, run.tocTotal, run.headings)
	}

	applyInternalLinks(run.doc, run.bodies, run.tocTotal)
}

func (p *pdfPipeline) assembleDocument() error {
	run := p.run
	plan, err := newPagePlan(run.tocs, run.bodies, run.req.Global.Copies, run.req.Global.Collate)

	if err != nil {
		return err
	}

	run.plan = plan
	if run.req.Global.Title != "" {
		run.doc.SetInfo("Title", run.req.Global.Title)
	}

	run.doc.SetInfo("Producer", "gowkhtmltopdf")
	run.doc.SetCompression(run.req.Global.UseCompression)
	run.doc.SetGrayscale(run.req.Global.Grayscale)
	run.doc.SetCreationTime(run.req.now())

	return nil
}

func (p *pdfPipeline) assembleCopies() error {
	run := p.run
	if run.plan.copies <= 1 {
		return nil
	}

	if err := materializeCopies(run.doc, run.plan.Ranges(), run.plan.copies); err != nil {
		return err
	}

	if run.plan.collate {
		return nil
	}

	if err := run.doc.ReorderPages(nonCollateOrder(run.plan.Ranges(), run.plan.copies)); err != nil {
		return fmt.Errorf("assemble copies: %w", err)
	}

	return nil
}

func (p *pdfPipeline) assembleHeadersFooters(ctx context.Context) error {
	run := p.run
	hfResult := drawHeadersFootersResult(ctx, run.loader, run.font, run.doc, run.req, run.plan, run.headings, run.log)

	if err := hfResult.Err(); err != nil {
		return fmt.Errorf("header/footer: %w", err)
	}

	return nil
}

func (p *pdfPipeline) Finalize(_ context.Context) error {
	run := p.run
	run.report("Done", progressComplete)

	if err := run.doc.Write(run.req.Output); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}
