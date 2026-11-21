// Package fonts udostępnia krój dołączony do modułu. Renderer mierzy nim tekst, gdy wywołujący
// nie poda własnego fontu; ten sam plik trzeba dać programowi, który rysuje SVG (przeglądarce
// przez @font-face, konwerterowi PDF jako font do osadzenia), inaczej tekst narysuje krój
// zastępczy o innych szerokościach.
//
// Recursive jest dziełem The Recursive Project Authors, na licencji SIL Open Font License 1.1
// (plik OFL.txt). Plik fontu jest niezmieniony.
package fonts

import _ "embed"

// RecursiveFamily to nazwa kroju wpisywana do atrybutu font-family.
const RecursiveFamily = "Recursive"

//go:embed Recursive-Regular.ttf
var recursive []byte

//go:embed OFL.txt
var license string

// Recursive zwraca kopię pliku fontu Recursive (odmiana zwykła, TrueType).
func Recursive() []byte { return append([]byte(nil), recursive...) }

// RecursiveLicense zwraca tekst licencji fontu; trzeba go dołączyć, rozpowszechniając font.
func RecursiveLicense() string { return license }
