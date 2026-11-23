package fonts

import (
	"bytes"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/fontdata"
)

// Kod modułu czyta kroje bez kopiowania (internal/fontdata); dane muszą być te same co kopie
// zwracane wywołującym.
func TestInternalDataMatchesCopies(t *testing.T) {
	for name, pair := range map[string][2][]byte{
		"Arimo": {fontdata.Arimo, Arimo()}, "ArimoBold": {fontdata.ArimoBold, ArimoBold()},
		"ArimoItalic": {fontdata.ArimoItalic, ArimoItalic()}, "NotoEmoji": {fontdata.NotoEmoji, NotoEmoji()},
	} {
		if len(pair[0]) == 0 || !bytes.Equal(pair[0], pair[1]) {
			t.Errorf("%s: internal data differs from the copy", name)
		}
	}
}
