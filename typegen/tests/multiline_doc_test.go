package typegen

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	gots "github.com/starius/api2/typegen"
	"github.com/starius/api2/typegen/tests/types"
)

// Multi-line godoc on an enum or named slice type must not leak continuation
// lines into gen.ts uncommented (the printer prefixes `// ` only once).
func TestMultilineDoc_enum_collapsed(t *testing.T) {
	assertNoUncommentedContinuation(t, reflect.TypeOf(types.MultilineEnum("")), "MultilineEnum")
}

func TestMultilineDoc_namedSlice_collapsed(t *testing.T) {
	assertNoUncommentedContinuation(t, reflect.TypeOf(types.MultilineArray{}), "MultilineArray")
}

func assertNoUncommentedContinuation(t *testing.T, seed reflect.Type, typeName string) {
	t.Helper()
	p := gots.NewFromTypes(seed)
	var buf bytes.Buffer
	gots.PrintTsTypes(p, &buf, func(t reflect.Type) string { return "" })
	out := buf.String()
	if !strings.Contains(out, typeName) {
		t.Fatalf("expected %q in output, got:\n%s", typeName, out)
	}
	needle := "The second line must not land uncommented in gen.ts."
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, needle) && !strings.HasPrefix(strings.TrimSpace(line), "//") {
			t.Fatalf("continuation line leaked uncommented into TypeScript output:\n%s", out)
		}
	}
}
