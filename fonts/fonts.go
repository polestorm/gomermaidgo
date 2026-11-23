// Package fonts udostępnia kroje dołączone do modułu. Renderer mierzy nimi tekst, gdy wywołujący
// nie poda własnego fontu; PDF i PNG rysują nimi litery.
//
// Krojem domyślnym jest Arimo, dzieło The Arimo Project Authors, na licencji SIL Open Font
// License 1.1 (plik OFL-Arimo.txt); pliki są niezmienione. Arimo ma te same szerokości znaków co
// Arial i Liberation Sans, więc rysunek SVG, który ich użyje zamiast Arimo (Arimo rzadko jest
// zainstalowany), zachowuje zmierzony układ.
//
// Krojem zapasowym dla znaków, których Arimo nie ma, jest Noto Emoji (jednobarwne emoji), dzieło
// Google LLC na tej samej licencji (plik OFL-NotoEmoji.txt). Plik jest zmieniony: to odmiana
// domyślna kroju zmiennego, bez tabel odmian (tools/staticfont).
package fonts

import (
	_ "embed"

	"github.com/polestorm/gomermaidgo/internal/fontdata"
)

// ArimoFamily to nazwa kroju domyślnego.
const ArimoFamily = "Arimo"

// DefaultFamily to wartość atrybutu font-family rysunku SVG mierzonego krojem domyślnym: Arimo,
// a po nim kroje o tych samych szerokościach, zainstalowane w Windows, macOS i większości
// dystrybucji Linuksa.
const DefaultFamily = "Arimo, Arial, 'Liberation Sans', sans-serif"

//go:embed Arimo-Regular.ttf
var arimo []byte

//go:embed Arimo-Bold.ttf
var arimoBold []byte

//go:embed Arimo-Italic.ttf
var arimoItalic []byte

//go:embed OFL-Arimo.txt
var license string

// Arimo zwraca kopię odmiany zwykłej kroju domyślnego (TrueType).
func Arimo() []byte { return append([]byte(nil), arimo...) }

// ArimoBold zwraca kopię odmiany pogrubionej; rysuje teksty z atrybutem font-weight="bold".
func ArimoBold() []byte { return append([]byte(nil), arimoBold...) }

// ArimoItalic zwraca kopię odmiany pochylonej; rysuje teksty z atrybutem font-style="italic".
func ArimoItalic() []byte { return append([]byte(nil), arimoItalic...) }

// ArimoLicense zwraca tekst licencji kroju domyślnego; trzeba go dołączyć, rozpowszechniając krój.
func ArimoLicense() string { return license }

// EmojiFamily to nazwa kroju zapasowego wpisywana do atrybutu font-family.
const EmojiFamily = "Noto Emoji"

//go:embed NotoEmoji-Regular.ttf
var notoEmoji []byte

//go:embed OFL-NotoEmoji.txt
var emojiLicense string

// NotoEmoji zwraca kopię kroju zapasowego z jednobarwnymi emoji (TrueType).
func NotoEmoji() []byte { return append([]byte(nil), notoEmoji...) }

// NotoEmojiLicense zwraca tekst licencji kroju zapasowego.
func NotoEmojiLicense() string { return emojiLicense }

func init() {
	fontdata.Arimo, fontdata.ArimoBold, fontdata.ArimoItalic, fontdata.NotoEmoji = arimo, arimoBold, arimoItalic, notoEmoji
}
