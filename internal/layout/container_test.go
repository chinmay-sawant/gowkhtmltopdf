package layout

import (
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

func TestContainerPropsParsed(t *testing.T) {
	s := sheet(t, `
		.a { container-type: inline-size; container-name: card }
		.b { container: sidebar / size }
		.c { container-type: normal }
	`)
	root := mustParse(t, `<html><body>
		<div class="a" id="a"></div>
		<div class="b" id="b"></div>
		<div class="c" id="c"></div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{s}, "print", testViewport, 800)
	byID := map[string]*html.Node{}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if id := n.Attribute("id"); id != "" {
				byID[id] = n
			}
		}

		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)

	if st := styles[byID["a"]]; st.ContainerType != "inline-size" || st.ContainerName != "card" {
		t.Errorf("a: type=%q name=%q", st.ContainerType, st.ContainerName)
	}

	if st := styles[byID["b"]]; st.ContainerType != "size" || st.ContainerName != "sidebar" {
		t.Errorf("b: type=%q name=%q", st.ContainerType, st.ContainerName)
	}

	if st := styles[byID["c"]]; st.ContainerType != "normal" {
		t.Errorf("c: type=%q", st.ContainerType)
	}
}

func TestContainerQueryNamedInlineSize(t *testing.T) {
	// 12pt font → 20em = 240pt. Wide card 400px=300pt matches; narrow 100px=75pt does not.
	s := sheet(t, `
		.card { container: card / inline-size; font-size: 12pt }
		.wide { width: 400px }
		.narrow { width: 100px }
		@container card (inline-size > 20em) {
			.title { color: red; font-size: 24pt }
		}
	`)
	root := mustParse(t, `<html><body>
		<div class="card wide"><p class="title" id="w">Wide</p></div>
		<div class="card narrow"><p class="title" id="n">Narrow</p></div>
	</body></html>`)
	cinfo := measureSizeContainers(root, resolveStyles(root, []*css.Stylesheet{s}, "print", testViewport, 800), testViewport)
	styles := resolveStylesWithContainers(root, []*css.Stylesheet{s}, "print", testViewport, 800, cinfo)

	byID := map[string]*html.Node{}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if id := n.Attribute("id"); id != "" {
				byID[id] = n
			}
		}

		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)

	stW := styles[byID["w"]]
	stN := styles[byID["n"]]

	if stW.Color[0] < 0.9 {
		t.Errorf("wide title color = %v, want red (container matched)", stW.Color)
	}

	if stW.FontSize < 23 || stW.FontSize > 25 {
		t.Errorf("wide title font-size = %v, want 24pt", stW.FontSize)
	}

	if stN.Color[0] > 0.1 {
		t.Errorf("narrow title color = %v, want black (no match)", stN.Color)
	}

	if stN.FontSize > 13 {
		t.Errorf("narrow title font-size = %v, want ~12pt", stN.FontSize)
	}
}

func TestContainerQueryUnnamedAndOrNot(t *testing.T) {
	s := sheet(t, `
		.box { container-type: inline-size; width: 300px; font-size: 12pt }
		@container (width > 200px) {
			.a { color: red }
		}
		@container (width > 500px) or (inline-size > 100px) {
			.b { color: blue }
		}
		@container not (inline-size < 50px) {
			.c { color: lime }
		}
		@container (min-width: 400px) {
			.d { color: yellow }
		}
	`)
	root := mustParse(t, `<html><body>
		<div class="box">
			<span class="a" id="a">a</span>
			<span class="b" id="b">b</span>
			<span class="c" id="c">c</span>
			<span class="d" id="d">d</span>
		</div>
	</body></html>`)
	pass1 := resolveStyles(root, []*css.Stylesheet{s}, "print", testViewport, 800)
	cinfo := measureSizeContainers(root, pass1, testViewport)
	styles := resolveStylesWithContainers(root, []*css.Stylesheet{s}, "print", testViewport, 800, cinfo)
	byID := map[string]*html.Node{}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if id := n.Attribute("id"); id != "" {
				byID[id] = n
			}
		}

		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)

	if styles[byID["a"]].Color[0] < 0.9 {
		t.Errorf("a: want red from (width > 200px)")
	}

	if styles[byID["b"]].Color[2] < 0.9 {
		t.Errorf("b: want blue from or-branch")
	}

	if styles[byID["c"]].Color[1] < 0.9 {
		t.Errorf("c: want green from not")
	}

	if styles[byID["d"]].Color[0] > 0.1 || styles[byID["d"]].Color[1] > 0.1 {
		t.Errorf("d: min-width:400px should not match 300px box, color=%v", styles[byID["d"]].Color)
	}
}

func TestContainerQueryRequiresContainment(t *testing.T) {
	// Name without container-type: not a size container; query must not apply.
	s := sheet(t, `
		.named { container-name: card; width: 400px }
		@container card (inline-size > 10px) {
			.x { color: red }
		}
	`)
	root := mustParse(t, `<html><body>
		<div class="named"><span class="x" id="x">x</span></div>
	</body></html>`)
	pass1 := resolveStyles(root, []*css.Stylesheet{s}, "print", testViewport, 800)

	cinfo := measureSizeContainers(root, pass1, testViewport)
	if len(cinfo) != 0 {
		t.Fatalf("expected no size containers, got %d", len(cinfo))
	}

	styles := resolveStylesWithContainers(root, []*css.Stylesheet{s}, "print", testViewport, 800, cinfo)

	var x *html.Node

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Attribute("id") == "x" {
			x = n
		}

		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)

	if styles[x].Color[0] > 0.1 {
		t.Errorf("without containment, @container must not apply; color=%v", styles[x].Color)
	}
}

func TestContainerQueryLayoutSwitch(t *testing.T) {
	s := sheet(t, `
		.card { container: card / inline-size; width: 400px; font-size: 12pt }
		@container card (inline-size > 20em) {
			.title { color: red }
		}
	`)
	res := layoutHTML(t, `<html><body>
		<div class="card"><p class="title">Hello</p></div>
	</body></html>`, s)

	texts := opsOfKind(res, OpText)
	if len(texts) == 0 {
		t.Fatal("no text ops")
	}

	found := false

	for _, op := range texts {
		if op.R > 0.9 && op.G < 0.1 && op.B < 0.1 {
			found = true
		}
	}

	if !found {
		t.Errorf("expected red text from @container match, texts=%+v", texts)
	}
}

func TestContainerQueryNearestNamedWins(t *testing.T) {
	s := sheet(t, `
		.outer { container: outer / inline-size; width: 400px; font-size: 12pt }
		.inner { container: inner / inline-size; width: 50px; font-size: 12pt }
		@container outer (inline-size > 20em) {
			.t { color: red }
		}
		@container inner (inline-size > 20em) {
			.t { color: blue }
		}
	`)
	root := mustParse(t, `<html><body>
		<div class="outer"><div class="inner"><span class="t" id="t">t</span></div></div>
	</body></html>`)
	pass1 := resolveStyles(root, []*css.Stylesheet{s}, "print", testViewport, 800)
	cinfo := measureSizeContainers(root, pass1, testViewport)
	styles := resolveStylesWithContainers(root, []*css.Stylesheet{s}, "print", testViewport, 800, cinfo)

	var tNode *html.Node

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Attribute("id") == "t" {
			tNode = n
		}

		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)

	st := styles[tNode]
	// Nearest named "inner" is too narrow → blue rule fails; "outer" matches → red.
	// But for @container outer, nearest ancestor named "outer" is the outer div (skipping
	// inner which has a different name). So red should apply.
	if st.Color[0] < 0.9 {
		t.Errorf("want red from @container outer; color=%v", st.Color)
	}

	if st.Color[2] > 0.1 {
		t.Errorf("inner query should not match; color=%v", st.Color)
	}
}
