//nolint:testpackage // parity checks the private descriptor registry.
package settings

import "testing"

func TestGlobalKeyDescriptorsHaveSetAndGetSides(t *testing.T) {
	t.Parallel()

	global := DefaultPdfGlobal()

	if len(globalKeys) < 50 {
		t.Fatalf("global key table has %d entries, want at least 50", len(globalKeys))
	}

	for name, descriptor := range globalKeys {
		if descriptor.apply == nil {
			t.Errorf("global key %q has no setter", name)
		}

		if descriptor.get == nil {
			t.Errorf("global key %q has no getter", name)
		}

		if _, ok := descriptor.get(&global); !ok {
			t.Errorf("global key %q getter does not expose a scalar value", name)
		}
	}
}
