// Package fonts udostępnia krój dołączony do modułu. Renderer mierzy nim tekst, gdy wywołujący
// nie poda własnego fontu; ten sam plik trzeba dać programowi, który rysuje SVG (przeglądarce
// przez @font-face, konwerterowi PDF jako font do osadzenia), inaczej tekst narysuje krój
// zastępczy o innych szerokościach.
//
// Recursive jest dziełem The Recursive Project Authors, na licencji SIL Open Font License 1.1
// (plik OFL.txt). Plik fontu jest niezmieniony.
//
// Krojem zapasowym dla znaków, których Recursive nie ma, jest Noto Emoji (jednobarwne emoji),
// dzieło Google LLC na tej samej licencji (plik OFL-NotoEmoji.txt). Plik jest zmieniony: to
// odmiana domyślna kroju zmiennego, bez tabel odmian (tools/staticfont).
package fonts

import _ "embed"

// RecursiveFamily to nazwa kroju wpisywana do atrybutu font-family.
const RecursiveFamily = "Recursive"

//go:embed Recursive-Regular.ttf
var recursive []byte

//go:embed Recursive-Bold.ttf
var recursiveBold []byte

//go:embed Recursive-Italic.ttf
var recursiveItalic []byte

// RecursiveBold zwraca kopię odmiany pogrubionej; rysuje teksty z atrybutem font-weight="bold".
func RecursiveBold() []byte { return append([]byte(nil), recursiveBold...) }

// RecursiveItalic zwraca kopię odmiany pochylonej; rysuje teksty z atrybutem font-style="italic".
func RecursiveItalic() []byte { return append([]byte(nil), recursiveItalic...) }

//go:embed OFL.txt
var license string

// Recursive zwraca kopię pliku fontu Recursive (odmiana zwykła, TrueType).
func Recursive() []byte { return append([]byte(nil), recursive...) }

// RecursiveLicense zwraca tekst licencji fontu; trzeba go dołączyć, rozpowszechniając font.
func RecursiveLicense() string { return license }

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
