// Package fontdata udostępnia kodowi modułu bajty krojów dołączonych bez kopiowania. Publiczny
// pakiet fonts zwraca kopie (wywołujący nie może zmienić danych osadzonych w programie), a kopia
// czterech plików kosztowała przy każdym uruchomieniu programu około dwóch milisekund. Pakiet fonts
// wypełnia te zmienne przy inicjalizacji; kod modułu tylko je czyta.
package fontdata

// Bajty krojów dołączonych (TrueType); tylko do odczytu.
var Arimo, ArimoBold, ArimoItalic, NotoEmoji []byte
