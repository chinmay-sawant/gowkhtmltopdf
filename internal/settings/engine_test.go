package settings

import (
	"errors"
	"testing"
)

func TestParseEngine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "", want: EngineNative},
		{in: "native", want: EngineNative},
		{in: "NATIVE", want: EngineNative},
		{in: " blinkless ", want: EngineBlinkless},
		{in: "Blinkless", want: EngineBlinkless},
		{in: "webkit", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.in, func(t *testing.T) {
			t.Parallel()

			got, err := ParseEngine(test.in)

			if test.wantErr {
				if !errors.Is(err, ErrInvalidEngine) {
					t.Fatalf("err = %v, want ErrInvalidEngine", err)
				}

				return
			}

			if err != nil {
				t.Fatalf("ParseEngine(%q): %v", test.in, err)
			}

			if got != test.want {
				t.Fatalf("ParseEngine(%q) = %q, want %q", test.in, got, test.want)
			}
		})
	}
}

func TestGlobalSetEngine(t *testing.T) {
	t.Parallel()

	global := DefaultPdfGlobal()

	if global.Engine != EngineNative {
		t.Fatalf("default engine = %q, want %q", global.Engine, EngineNative)
	}

	if err := global.Set("engine", "blinkless"); err != nil {
		t.Fatalf("Set(engine): %v", err)
	}

	if global.Engine != EngineBlinkless {
		t.Fatalf("engine = %q, want %q", global.Engine, EngineBlinkless)
	}

	if err := global.Set("engine", "trident"); !errors.Is(err, ErrInvalidEngine) {
		t.Fatalf("Set(engine, trident) err = %v, want ErrInvalidEngine", err)
	}

	if global.Engine != EngineBlinkless {
		t.Fatalf("invalid Set changed engine to %q", global.Engine)
	}
}
