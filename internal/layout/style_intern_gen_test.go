package layout

import (
	"math"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"
)

// TestStyleInternGeneratedFieldsComplete fails when ResolvedStyle gains or
// loses a field without regenerating style_intern_gen.go. That is the guard
// against an equality or fingerprint that silently ignores a new field.
func TestStyleInternGeneratedFieldsComplete(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeOf(ResolvedStyle{})
	live := make([]string, 0, typ.NumField())

	for i := range typ.NumField() {
		live = append(live, typ.Field(i).Name)
	}

	generated := styleInternFields()
	if !slices.Equal(live, generated) {
		t.Fatalf("generated field list is stale:\nlive      %v\ngenerated %v", live, generated)
	}
}

// TestStyleInternGeneratedEqualityMatchesReflect compares the generated
// equality with reflect.DeepEqual on random styles, a same-content copy, and a
// single-field mutation.
func TestStyleInternGeneratedEqualityMatchesReflect(t *testing.T) {
	t.Parallel()

	//nolint:gosec // PCG with a fixed seed is for deterministic tests, not security.
	rng := rand.New(rand.NewPCG(1, 2))

	for range 2000 {
		original := styleInternRandomStyle(rng)
		mutated := original

		assertStyleInternAgreement(t, "value copy", &original, &mutated)

		styleInternMutateOneField(rng, &mutated)
		assertStyleInternAgreement(t, "single-field mutation", &original, &mutated)
	}
}

// TestStyleInternGeneratedEqualityComparesContents proves slice and map
// equality is by content, not backing pointer, and that equal styles share a
// fingerprint.
func TestStyleInternGeneratedEqualityComparesContents(t *testing.T) {
	t.Parallel()

	left := initialStyle()
	left.FontFamily = []string{"Helvetica", "sans-serif"}
	left.StrokeDashArray = []float64{3, 1.5}
	left.CustomProps = map[string]string{"--x": "1", "--y": "var(--x)"}

	right := left
	right.FontFamily = []string{"Helvetica", "sans-serif"}
	right.StrokeDashArray = []float64{3, 1.5}
	right.CustomProps = map[string]string{"--x": "1", "--y": "var(--x)"}

	if !styleInternEqual(&left, &right) {
		t.Fatal("equal slice and map contents must compare equal")
	}

	if styleInternFingerprint(&left) != styleInternFingerprint(&right) {
		t.Fatal("equal slice and map contents must share a fingerprint")
	}

	right.StrokeDashArray[0] = 4
	if styleInternEqual(&left, &right) {
		t.Fatal("different dash contents must not compare equal")
	}
}

// TestStyleInternGeneratedFingerprintNormalizesZero pins the -0.0 case:
// -0.0 == 0.0, so the two must hash identically or interning would miss the
// share.
func TestStyleInternGeneratedFingerprintNormalizesZero(t *testing.T) {
	t.Parallel()

	positive := initialStyle()
	positive.FontSize = 0
	negative := positive
	negative.FontSize = math.Copysign(0, -1)

	if !styleInternEqual(&positive, &negative) {
		t.Fatal("-0.0 and 0.0 must compare equal")
	}

	if styleInternFingerprint(&positive) != styleInternFingerprint(&negative) {
		t.Fatalf(
			"-0.0 fingerprint %d != 0.0 fingerprint %d",
			styleInternFingerprint(&negative), styleInternFingerprint(&positive),
		)
	}
}

// TestStyleInternGeneratedNaNMatchesReflect pins NaN semantics: NaN == NaN is
// false, and the generated equality must agree with reflect.DeepEqual.
func TestStyleInternGeneratedNaNMatchesReflect(t *testing.T) {
	t.Parallel()

	original := initialStyle()
	original.Top = math.NaN()
	copied := original

	if styleInternEqual(&original, &copied) {
		t.Fatal("NaN fields must not compare equal")
	}

	if reflect.DeepEqual(original, copied) {
		t.Fatal("sanity: reflect.DeepEqual must also reject NaN fields")
	}
}

// assertStyleInternAgreement requires the generated equality to agree with
// reflect.DeepEqual and equal styles to share a fingerprint.
func assertStyleInternAgreement(t *testing.T, label string, left, right *ResolvedStyle) {
	t.Helper()

	generated := styleInternEqual(left, right)
	reflected := reflect.DeepEqual(*left, *right)

	if generated != reflected {
		t.Errorf("%s: styleInternEqual=%v reflect.DeepEqual=%v", label, generated, reflected)
	}

	if generated && styleInternFingerprint(left) != styleInternFingerprint(right) {
		t.Errorf(
			"%s: equal styles fingerprint differently (%d vs %d)",
			label, styleInternFingerprint(left), styleInternFingerprint(right),
		)
	}
}

// styleInternRandomStyle fills every exported field and famHash with random
// values. NaN is excluded here so a single NaN field cannot mask a missed
// comparison in the mutation agreement check; NaN has its own test.
func styleInternRandomStyle(rng *rand.Rand) ResolvedStyle {
	var style ResolvedStyle

	fillStyleInternStruct(rng, reflect.ValueOf(&style).Elem())
	style.famHash = rng.Uint64()

	return style
}

func fillStyleInternStruct(rng *rand.Rand, value reflect.Value) {
	typ := value.Type()

	for i := range typ.NumField() {
		field := value.Field(i)
		if !field.CanSet() {
			continue
		}

		fillStyleInternValue(rng, field)
	}
}

// fillStyleInternValue dispatches one reflect value to the filler for its
// kind. The default branch panics so a new ResolvedStyle field kind fails
// loudly instead of being silently skipped.
func fillStyleInternValue(rng *rand.Rand, value reflect.Value) {
	switch value.Kind() { //nolint:exhaustive // default panics; a new kind must fail the test loudly.
	case reflect.String, reflect.Bool, reflect.Int, reflect.Uint64, reflect.Float64:
		fillStyleInternScalar(rng, value)
	case reflect.Array, reflect.Slice:
		fillStyleInternSequence(rng, value)
	case reflect.Map:
		fillStyleInternMap(rng, value)
	case reflect.Struct:
		fillStyleInternStruct(rng, value)
	default:
		panic("style intern randomizer: unhandled kind " + value.Kind().String())
	}
}

func fillStyleInternScalar(rng *rand.Rand, value reflect.Value) {
	switch value.Kind() { //nolint:exhaustive // caller filters kinds; default signals a bug.
	case reflect.String:
		value.SetString(styleInternRandomString(rng))
	case reflect.Bool:
		value.SetBool(rng.IntN(2) == 0)
	case reflect.Int:
		value.SetInt(int64(rng.IntN(5) - 2))
	case reflect.Uint64:
		value.SetUint(rng.Uint64())
	case reflect.Float64:
		value.SetFloat(styleInternRandomFloat(rng))
	default:
		panic("style intern randomizer: unhandled scalar kind " + value.Kind().String())
	}
}

func fillStyleInternSequence(rng *rand.Rand, value reflect.Value) {
	if value.Kind() == reflect.Array {
		for i := range value.Len() {
			fillStyleInternValue(rng, value.Index(i))
		}

		return
	}

	length := rng.IntN(4)
	value.Set(reflect.MakeSlice(value.Type(), length, length))

	for i := range length {
		fillStyleInternValue(rng, value.Index(i))
	}
}

func fillStyleInternMap(rng *rand.Rand, value reflect.Value) {
	styleMap := reflect.MakeMap(value.Type())

	for range rng.IntN(3) {
		styleMap.SetMapIndex(
			reflect.ValueOf(styleInternRandomString(rng)),
			reflect.ValueOf(styleInternRandomString(rng)),
		)
	}

	value.Set(styleMap)
}

// styleInternMutateOneField changes one random settable field, including the
// unexported famHash.
func styleInternMutateOneField(rng *rand.Rand, style *ResolvedStyle) {
	value := reflect.ValueOf(style).Elem()
	typ := value.Type()
	index := rng.IntN(typ.NumField())
	field := value.Field(index)

	if field.CanSet() {
		fillStyleInternValue(rng, field)

		return
	}

	if typ.Field(index).Name == "famHash" {
		style.famHash = rng.Uint64() + 1

		return
	}

	panic("style intern randomizer: field " + typ.Field(index).Name + " is not settable")
}

func styleInternRandomString(rng *rand.Rand) string {
	switch rng.IntN(5) {
	case 0:
		return ""
	case 1:
		return "a"
	case 2:
		return "ab"
	case 3:
		return "display"
	default:
		return "border-top"
	}
}

func styleInternRandomFloat(rng *rand.Rand) float64 {
	switch rng.IntN(7) {
	case 0:
		return 0
	case 1:
		return math.Copysign(0, -1)
	case 2:
		return 1
	case 3:
		return -1.5
	case 4:
		return 3.25
	case 5:
		return math.Inf(1)
	default:
		return math.Inf(-1)
	}
}
