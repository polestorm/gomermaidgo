package font

// GlyphID zwraca numer glifu znaku; 0 oznacza glif zastępczy (znaku nie ma w kroju).
func (f *Face) GlyphID(c rune) uint32 { return f.glyph(c) }
