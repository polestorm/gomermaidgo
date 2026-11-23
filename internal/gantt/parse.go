// Package gantt obsługuje wykres Gantta: tekst → model → scena.
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package gantt

import (
	"strconv"
	"strings"
	"time"

	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// Kody błędów składni wykresu Gantta.
const (
	CodeHeader      = "gantt-header"
	CodeUnsupported = "gantt-unsupported"
	CodeStatement   = "gantt-statement"
	CodeTask        = "gantt-task"
	CodeDate        = "gantt-date"
	CodeEmpty       = "gantt-empty"
	maxTasks        = 500
	maxSpanDays     = 36600 // sto lat: chroni podziałkę przed datą z literówką
)

// Diagram to rozebrany wykres Gantta.
type Diagram struct {
	Title       string
	AccTitle    string
	AccDescr    string
	AxisFormat  string // zapis dat na podziałce, w notacji strftime
	TickEvery   int    // odstęp podziałki; 0 = dobierany do rozpiętości
	TickUnit    string // "day", "week", "month" albo "year"
	Sections    []string
	Tasks       []*Task
	Excluded    func(day time.Time) bool // dzień wyłączony z pracy
	HasExcludes bool
}

// Task to zadanie albo kamień milowy.
type Task struct {
	Name      string
	ID        string
	Section   int // indeks w Diagram.Sections; -1 = poza sekcjami
	Start     time.Time
	End       time.Time
	Active    bool
	Done      bool
	Critical  bool
	Milestone bool
	Line      int

	// Zapis ze źródła, rozstrzygany po wczytaniu wszystkich zadań.
	startSpec, endSpec string
}

var (
	durationSpec = syntax.Lazy(`^(\d+(?:\.\d+)?)(ms|s|m|h|d|w|M|y)$`)
	tickSpec     = syntax.Lazy(`^(\d+)(day|week|month|year)$`)
	weekdays     = map[string]time.Weekday{"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday,
		"wednesday": time.Wednesday, "thursday": time.Thursday, "friday": time.Friday, "saturday": time.Saturday}
	tags = map[string]bool{"active": true, "done": true, "crit": true, "milestone": true}
)

type parser struct {
	d          *Diagram
	acc        syntax.Accessibility
	layout     string // zapis dat w notacji języka Go
	hasTime    bool   // zapis dat zawiera godzinę
	inclusive  bool   // daty końca są włącznie
	weekend    [2]time.Weekday
	noWeekends bool
	noDays     map[time.Weekday]bool
	noDates    map[string]bool
	section    int
	ids        map[string]*Task
}

// goLayout zamienia zapis dat Mermaid (YYYY, MM, DD, HH, mm, ss) na zapis języka Go.
func goLayout(format string) (layout string, hasTime bool, ok bool) {
	tokens := []struct{ from, to string }{
		{"YYYY", "2006"}, {"YY", "06"}, {"MM", "01"}, {"M", "1"}, {"DD", "02"}, {"D", "2"},
		{"HH", "15"}, {"mm", "04"}, {"ss", "05"},
	}
	var b strings.Builder
	seen := map[string]bool{}
	for i := 0; i < len(format); {
		matched := false
		for _, t := range tokens {
			if strings.HasPrefix(format[i:], t.from) {
				b.WriteString(t.to)
				i += len(t.from)
				seen[t.from[:1]] = true
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		c := format[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			return "", false, false // nieznany znacznik albo cyfra, którą Go wziąłby za część wzorca
		}
		b.WriteByte(c)
		i++
	}
	if !seen["Y"] || !(seen["M"] && seen["D"]) {
		return "", false, false
	}
	return b.String(), seen["H"], true
}

// Parse rozbiera źródło wykresu Gantta.
func Parse(source string) (*Diagram, error) {
	p := &parser{d: &Diagram{AxisFormat: "%Y-%m-%d"}, layout: "2006-01-02", section: -1,
		weekend: [2]time.Weekday{time.Saturday, time.Sunday},
		noDays:  map[time.Weekday]bool{}, noDates: map[string]bool{}, ids: map[string]*Task{}}
	seenHeader := false
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for n, raw := range strings.Split(source, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if s != "gantt" {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"gantt\", found %q", s)
			}
			seenHeader = true
			continue
		}
		if p.acc.Line(s) {
			continue
		}
		if err := p.statement(s, line); err != nil {
			return nil, err
		}
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"gantt\"")
	}
	if p.acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	if len(p.d.Tasks) == 0 {
		return nil, syntax.Errorf(0, CodeEmpty, "chart has no tasks")
	}
	p.d.AccTitle, p.d.AccDescr = p.acc.Title, p.acc.Descr
	p.d.Excluded = p.excluded
	p.d.HasExcludes = p.noWeekends || len(p.noDays) > 0 || len(p.noDates) > 0
	if err := p.resolve(); err != nil {
		return nil, err
	}
	return p.d, nil
}

func (p *parser) excluded(day time.Time) bool {
	wd := day.Weekday()
	return p.noDays[wd] || (p.noWeekends && (wd == p.weekend[0] || wd == p.weekend[1])) || p.noDates[day.Format("2006-01-02")]
}

func (p *parser) statement(s string, line int) error {
	word, rest := s, ""
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		word, rest = s[:i], strings.TrimSpace(s[i+1:])
	}
	switch word {
	case "title":
		p.d.Title = rest
	case "dateFormat":
		layout, hasTime, ok := goLayout(rest)
		if !ok {
			return syntax.Errorf(line, CodeDate, "dateFormat %q is not supported; use YYYY, MM, DD and optionally HH, mm, ss", rest)
		}
		if len(p.d.Tasks) > 0 {
			return syntax.Errorf(line, CodeDate, "dateFormat must come before the first task")
		}
		p.layout, p.hasTime = layout, hasTime
	case "axisFormat":
		if rest == "" {
			return syntax.Errorf(line, CodeStatement, "axisFormat needs a format such as %%Y-%%m-%%d")
		}
		p.d.AxisFormat = rest
	case "tickInterval":
		m := tickSpec.FindStringSubmatch(rest)
		n := 0
		if m != nil {
			n, _ = strconv.Atoi(m[1])
		}
		if n < 1 || n > 1000 {
			return syntax.Errorf(line, CodeStatement, "tickInterval %q is not supported; use a whole number with day, week, month or year", rest)
		}
		p.d.TickEvery, p.d.TickUnit = n, m[2]
	case "excludes":
		for _, item := range strings.FieldsFunc(rest, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' }) {
			lower := strings.ToLower(item)
			if lower == "weekends" {
				p.noWeekends = true
			} else if wd, ok := weekdays[lower]; ok {
				p.noDays[wd] = true
			} else if t, err := time.Parse("2006-01-02", item); err == nil {
				p.noDates[t.Format("2006-01-02")] = true
			} else {
				return syntax.Errorf(line, CodeDate, "excludes takes \"weekends\", a weekday name or a date YYYY-MM-DD, found %q", item)
			}
		}
	case "weekend":
		switch strings.ToLower(rest) {
		case "friday":
			p.weekend = [2]time.Weekday{time.Friday, time.Saturday}
		case "saturday":
			p.weekend = [2]time.Weekday{time.Saturday, time.Sunday}
		default:
			return syntax.Errorf(line, CodeStatement, "weekend takes \"friday\" or \"saturday\", found %q", rest)
		}
	case "inclusiveEndDates":
		p.inclusive = true
	case "todayMarker", "topAxis", "click":
		// Bez znaczenia dla rysunku: znacznika „dziś” nie rysujemy nigdy (wynik nie może zależeć
		// od dnia uruchomienia), podziałka jest zawsze u góry, odnośników rysunek nie ma.
	case "section":
		if rest == "" {
			return syntax.Errorf(line, CodeStatement, "section needs a name")
		}
		p.d.Sections = append(p.d.Sections, rest)
		p.section = len(p.d.Sections) - 1
	case "includes", "displayMode", "vert":
		return syntax.Errorf(line, CodeUnsupported, "%q is outside the supported profile", word)
	default:
		return p.task(s, line)
	}
	return nil
}

func (p *parser) task(s string, line int) error {
	name, meta, ok := strings.Cut(s, ":")
	name = strings.TrimSpace(name)
	if !ok || name == "" {
		return syntax.Errorf(line, CodeStatement, "unrecognised statement %q; a task is written \"name : [tags,] [id,] [start,] end\"", s)
	}
	if len(p.d.Tasks) >= maxTasks {
		return syntax.Errorf(line, CodeTask, "too many tasks (limit %d)", maxTasks)
	}
	t := &Task{Name: name, Section: p.section, Line: line}
	var items []string
	for _, item := range strings.Split(meta, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	for len(items) > 0 && tags[items[0]] {
		switch items[0] {
		case "active":
			t.Active = true
		case "done":
			t.Done = true
		case "crit":
			t.Critical = true
		default:
			t.Milestone = true
		}
		items = items[1:]
	}
	switch len(items) {
	case 1:
		t.endSpec = items[0]
	case 2:
		t.startSpec, t.endSpec = items[0], items[1]
	case 3:
		t.ID, t.startSpec, t.endSpec = items[0], items[1], items[2]
	default:
		return syntax.Errorf(line, CodeTask, "task %q needs an end or a duration, optionally preceded by a start and an id", name)
	}
	if t.ID != "" {
		if p.ids[t.ID] != nil {
			return syntax.Errorf(line, CodeTask, "task id %q is already used", t.ID)
		}
		p.ids[t.ID] = t
	}
	p.d.Tasks = append(p.d.Tasks, t)
	return nil
}

func (p *parser) date(s string) (time.Time, bool) {
	t, err := time.Parse(p.layout, s)
	return t, err == nil
}

// addWork przesuwa chwilę o czas pracy: dni wyłączone się nie liczą.
func (p *parser) addWork(start time.Time, d time.Duration) time.Time {
	if !p.d.HasExcludes || d <= 0 {
		return start.Add(d)
	}
	t := start
	for guard := 0; d > 0 && guard < 100000; guard++ {
		dayStart := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		next := dayStart.AddDate(0, 0, 1)
		if p.excluded(dayStart) {
			t = next
			continue
		}
		if room := next.Sub(t); d <= room {
			return t.Add(d)
		} else {
			d -= room
			t = next
		}
	}
	return t
}

// resolve wyznacza początki i końce zadań; zadanie może odwoływać się do zadania zapisanego niżej.
func (p *parser) resolve() error {
	done := map[*Task]bool{}
	refs := func(spec, keyword string, line int) ([]*Task, bool, error) {
		rest, ok := strings.CutPrefix(spec, keyword+" ")
		if !ok {
			return nil, false, nil
		}
		var out []*Task
		for _, id := range strings.Fields(rest) {
			t := p.ids[id]
			if t == nil {
				return nil, true, syntax.Errorf(line, CodeTask, "%q refers to unknown task id %q", keyword, id)
			}
			out = append(out, t)
		}
		if len(out) == 0 {
			return nil, true, syntax.Errorf(line, CodeTask, "%q needs a task id", keyword)
		}
		return out, true, nil
	}
	try := func(i int, t *Task) (bool, error) {
		// Początek.
		switch after, isRef, err := refs(t.startSpec, "after", t.Line); {
		case err != nil:
			return false, err
		case isRef:
			for _, a := range after {
				if !done[a] {
					return false, nil
				}
				if a.End.After(t.Start) {
					t.Start = a.End
				}
			}
		case t.startSpec == "":
			if i > 0 {
				if !done[p.d.Tasks[i-1]] {
					return false, nil
				}
				t.Start = p.d.Tasks[i-1].End
			}
		default:
			d, ok := p.date(t.startSpec)
			if !ok {
				return false, syntax.Errorf(t.Line, CodeDate, "task %q: start %q is neither a date in the dateFormat nor \"after id\"", t.Name, t.startSpec)
			}
			t.Start = d
		}
		// Koniec.
		if m := durationSpec.FindStringSubmatch(t.endSpec); m != nil {
			n, _ := strconv.ParseFloat(m[1], 64)
			switch m[2] {
			case "M":
				t.End = t.Start.AddDate(0, int(n), 0)
			case "y":
				t.End = t.Start.AddDate(int(n), 0, 0)
			default:
				unit := map[string]time.Duration{"ms": time.Millisecond, "s": time.Second, "m": time.Minute, "h": time.Hour, "d": 24 * time.Hour, "w": 7 * 24 * time.Hour}[m[2]]
				t.End = p.addWork(t.Start, time.Duration(n*float64(unit)))
			}
			return true, nil
		}
		switch until, isRef, err := refs(t.endSpec, "until", t.Line); {
		case err != nil:
			return false, err
		case isRef:
			for k, u := range until {
				if !done[u] && u.startSpec != "" && !strings.HasPrefix(u.startSpec, "after ") {
					// Początek podany datą jest znany bez rozstrzygania całego zadania.
					if d, ok := p.date(u.startSpec); ok {
						u.Start = d
					}
				} else if !done[u] {
					return false, nil
				}
				if k == 0 || u.Start.Before(t.End) {
					t.End = u.Start
				}
			}
			return true, nil
		}
		d, ok := p.date(t.endSpec)
		if !ok {
			return false, syntax.Errorf(t.Line, CodeDate, "task %q: end %q is neither a date in the dateFormat, a duration such as 5d, nor \"until id\"", t.Name, t.endSpec)
		}
		if p.inclusive && !p.hasTime {
			d = d.AddDate(0, 0, 1)
		}
		t.End = d
		return true, nil
	}
	if first := p.d.Tasks[0]; first.startSpec == "" {
		return syntax.Errorf(first.Line, CodeDate, "the first task needs a start date")
	}
	for left := len(p.d.Tasks); left > 0; {
		progress := false
		for i, t := range p.d.Tasks {
			if done[t] {
				continue
			}
			ok, err := try(i, t)
			if err != nil {
				return err
			}
			if ok {
				done[t] = true
				left--
				progress = true
			}
		}
		if !progress {
			for _, t := range p.d.Tasks {
				if !done[t] {
					return syntax.Errorf(t.Line, CodeTask, "task %q depends on itself through \"after\" or \"until\"", t.Name)
				}
			}
		}
	}
	lo, hi := p.d.Tasks[0].Start, p.d.Tasks[0].End
	for _, t := range p.d.Tasks {
		if t.End.Before(t.Start) {
			return syntax.Errorf(t.Line, CodeDate, "task %q ends before it starts", t.Name)
		}
		if t.Start.Before(lo) {
			lo = t.Start
		}
		if t.End.After(hi) {
			hi = t.End
		}
	}
	if lo.Year() < 1900 || hi.Year() > 2200 || hi.Sub(lo) > maxSpanDays*24*time.Hour {
		return syntax.Errorf(0, CodeDate, "chart spans %s to %s; supported are years 1900–2200 and at most 100 years", lo.Format("2006-01-02"), hi.Format("2006-01-02"))
	}
	return nil
}

// Dump zapisuje wykres jako tekst w stałej kolejności; służy za plik wzorcowy parsera.
func (d *Diagram) Dump() string {
	var b strings.Builder
	b.WriteString("gantt")
	if d.Title != "" {
		b.WriteString(" title=" + strconv.Quote(d.Title))
	}
	b.WriteString(" axis=" + strconv.Quote(d.AxisFormat))
	if d.TickEvery > 0 {
		b.WriteString(" tick=" + strconv.Itoa(d.TickEvery) + d.TickUnit)
	}
	b.WriteByte('\n')
	const stamp = "2006-01-02T15:04"
	section := -1
	for _, t := range d.Tasks {
		if t.Section != section {
			section = t.Section
			b.WriteString("section " + strconv.Quote(d.Sections[section]) + "\n")
		}
		b.WriteString("task " + strconv.Quote(t.Name) + " " + t.Start.Format(stamp) + " " + t.End.Format(stamp))
		if t.ID != "" {
			b.WriteString(" id=" + t.ID)
		}
		for _, f := range []struct {
			on   bool
			name string
		}{{t.Active, "active"}, {t.Done, "done"}, {t.Critical, "crit"}, {t.Milestone, "milestone"}} {
			if f.on {
				b.WriteString(" " + f.name)
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}
