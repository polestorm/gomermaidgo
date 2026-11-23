package syntax

import (
	"regexp"
	"sync"
)

// Regexp to wyrażenie regularne kompilowane przy pierwszym użyciu. Pakiety rozbioru deklarują
// wzorce na poziomie pakietu; kompilowane od razu kosztowały przy każdym uruchomieniu programu
// kilka milisekund (wzorce z klasami Unicode są duże), choć rysunek potrzebuje wzorców jednego
// typu diagramu. Metody odpowiadają metodom regexp.Regexp.
type Regexp struct {
	src  string
	once sync.Once
	re   *regexp.Regexp
}

var (
	registryMu sync.Mutex
	registry   []*Regexp
)

// Lazy zwraca wyrażenie kompilowane przy pierwszym użyciu. Błąd wzorca ujawnia się wtedy, więc
// wszystkie wzorce są zapisywane w rejestrze, a test kompiluje je od razu (Patterns).
func Lazy(src string) *Regexp {
	r := &Regexp{src: src}
	registryMu.Lock()
	registry = append(registry, r)
	registryMu.Unlock()
	return r
}

// Patterns zwraca źródła wszystkich wzorców utworzonych przez Lazy (dla testów).
func Patterns() []string {
	registryMu.Lock()
	defer registryMu.Unlock()
	out := make([]string, len(registry))
	for i, r := range registry {
		out[i] = r.src
	}
	return out
}

// Re zwraca skompilowane wyrażenie.
func (r *Regexp) Re() *regexp.Regexp {
	r.once.Do(func() { r.re = regexp.MustCompile(r.src) })
	return r.re
}

func (r *Regexp) MatchString(s string) bool              { return r.Re().MatchString(s) }
func (r *Regexp) FindString(s string) string             { return r.Re().FindString(s) }
func (r *Regexp) FindStringIndex(s string) []int         { return r.Re().FindStringIndex(s) }
func (r *Regexp) FindStringSubmatch(s string) []string   { return r.Re().FindStringSubmatch(s) }
func (r *Regexp) FindStringSubmatchIndex(s string) []int { return r.Re().FindStringSubmatchIndex(s) }
func (r *Regexp) FindAllString(s string, n int) []string {
	return r.Re().FindAllString(s, n)
}
func (r *Regexp) FindAllStringIndex(s string, n int) [][]int {
	return r.Re().FindAllStringIndex(s, n)
}
func (r *Regexp) FindAllStringSubmatch(s string, n int) [][]string {
	return r.Re().FindAllStringSubmatch(s, n)
}
func (r *Regexp) FindAllStringSubmatchIndex(s string, n int) [][]int {
	return r.Re().FindAllStringSubmatchIndex(s, n)
}
func (r *Regexp) FindSubmatch(b []byte) [][]byte { return r.Re().FindSubmatch(b) }
func (r *Regexp) FindAllSubmatch(b []byte, n int) [][][]byte {
	return r.Re().FindAllSubmatch(b, n)
}
func (r *Regexp) ReplaceAllString(s, repl string) string { return r.Re().ReplaceAllString(s, repl) }
func (r *Regexp) ReplaceAllLiteralString(s, repl string) string {
	return r.Re().ReplaceAllLiteralString(s, repl)
}
func (r *Regexp) ReplaceAllStringFunc(s string, f func(string) string) string {
	return r.Re().ReplaceAllStringFunc(s, f)
}
func (r *Regexp) ReplaceAll(b, repl []byte) []byte { return r.Re().ReplaceAll(b, repl) }
func (r *Regexp) Split(s string, n int) []string   { return r.Re().Split(s, n) }
func (r *Regexp) SubexpNames() []string            { return r.Re().SubexpNames() }
func (r *Regexp) SubexpIndex(name string) int      { return r.Re().SubexpIndex(name) }
func (r *Regexp) String() string                   { return r.src }
