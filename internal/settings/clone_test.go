package settings //nolint:testpackage // Build clones unexported builder fields.

import "testing"

const (
	cloneAllowPath = "/tmp/a"
	cloneMutated   = "mutated"
)

func TestPdfGlobalOptionsBuildClonesAllowNetworkAndHeaderReplace(t *testing.T) {
	t.Parallel()

	options, err := NewPdfGlobalOptions().WithSetting("allow", cloneAllowPath)
	if err != nil {
		t.Fatalf("WithSetting(allow): %v", err)
	}

	options.global.Load.NetworkPolicySet = true
	options.global.Load.NetworkAllowedHosts = []string{"reports.example.test"}
	options.global.Load.NetworkAllowedSchemes = []string{"http"}
	options.global.Header.Replace = map[string]string{"name": "before"}
	options.global.Footer.Replace = map[string]string{"page": "before"}

	got := options.Build()

	if _, err := options.WithSetting("allow", "/tmp/mutated"); err != nil {
		t.Fatalf("mutate allow: %v", err)
	}

	options.global.Load.NetworkAllowedHosts[0] = "mutated.example.test"
	options.global.Header.Replace["name"] = "after"
	options.global.Footer.Replace["page"] = "after"

	if len(got.Load.Allow) != 1 || got.Load.Allow[0] != cloneAllowPath {
		t.Fatalf("Allow snapshot = %v, want [%s]", got.Load.Allow, cloneAllowPath)
	}

	if got.Load.NetworkAllowedHosts[0] != "reports.example.test" {
		t.Fatalf("NetworkAllowedHosts snapshot = %v", got.Load.NetworkAllowedHosts)
	}

	if got.Header.Replace["name"] != "before" || got.Footer.Replace["page"] != "before" {
		t.Fatalf("header/footer Replace snapshot changed: header=%v footer=%v",
			got.Header.Replace, got.Footer.Replace)
	}
}

func TestCloneHelpersCopyNestedSlicesAndMaps(t *testing.T) {
	t.Parallel()

	global := DefaultPdfGlobal()
	global.Load.Allow = []string{cloneAllowPath}
	global.Load.NetworkAllowedHosts = []string{"exact.example.test"}
	global.Header.Replace = map[string]string{"k": "v"}
	clonedGlobal := ClonePdfGlobal(global)
	global.Load.Allow[0] = cloneMutated
	global.Load.NetworkAllowedHosts[0] = cloneMutated
	global.Header.Replace["k"] = cloneMutated

	if clonedGlobal.Load.Allow[0] != cloneAllowPath {
		t.Fatalf("ClonePdfGlobal Allow = %v", clonedGlobal.Load.Allow)
	}

	if clonedGlobal.Load.NetworkAllowedHosts[0] != "exact.example.test" {
		t.Fatalf("ClonePdfGlobal hosts = %v", clonedGlobal.Load.NetworkAllowedHosts)
	}

	if clonedGlobal.Header.Replace["k"] != "v" {
		t.Fatalf("ClonePdfGlobal header replace = %v", clonedGlobal.Header.Replace)
	}

	obj := DefaultPdfObject()
	obj.Load.InlineHTML = []byte("<p>ok</p>")
	obj.Header.Replace = map[string]string{"k": "v"}
	clonedObj := ClonePdfObject(obj)
	obj.Load.InlineHTML[1] = 'X'
	obj.Header.Replace["k"] = cloneMutated

	if string(clonedObj.Load.InlineHTML) != "<p>ok</p>" {
		t.Fatalf("ClonePdfObject InlineHTML = %q", clonedObj.Load.InlineHTML)
	}

	if clonedObj.Header.Replace["k"] != "v" {
		t.Fatalf("ClonePdfObject header replace = %v", clonedObj.Header.Replace)
	}

	img := DefaultImageGlobal()
	img.Load.Allow = []string{cloneAllowPath}
	img.Load.NetworkAllowedHosts = []string{"img.example.test"}
	clonedImg := CloneImageGlobal(img)
	img.Load.Allow[0] = cloneMutated
	img.Load.NetworkAllowedHosts[0] = cloneMutated

	if clonedImg.Load.Allow[0] != cloneAllowPath || clonedImg.Load.NetworkAllowedHosts[0] != "img.example.test" {
		t.Fatalf("CloneImageGlobal = allow %v hosts %v", clonedImg.Load.Allow, clonedImg.Load.NetworkAllowedHosts)
	}
}
