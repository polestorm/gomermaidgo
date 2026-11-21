package gantt

import (
	"errors"
	"testing"
	"time"

	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: GANTT-005
func TestParseErrors(t *testing.T) {
	cases := []struct {
		name, src string
		line      int
		code      string
	}{
		{"empty", "", 0, CodeHeader},
		{"wrong header", "ganttChart\nA : 2026-01-01, 1d", 1, CodeHeader},
		{"no tasks", "gantt\ntitle Pusty", 0, CodeEmpty},
		{"not a task", "gantt\nzadanie bez dwukropka", 2, CodeStatement},
		{"task without end", "gantt\nA :", 2, CodeTask},
		{"too many fields", "gantt\nA : a, 2026-01-01, 1d, 2d", 2, CodeTask},
		{"first task without start", "gantt\nA : 3d", 2, CodeDate},
		{"bad start", "gantt\nA : wczoraj, 3d", 2, CodeDate},
		{"bad end", "gantt\nA : 2026-01-01, kiedyś", 2, CodeDate},
		{"date in another format", "gantt\ndateFormat DD.MM.YYYY\nA : 2026-01-01, 3d", 3, CodeDate},
		{"unknown id", "gantt\nA : a, 2026-01-01, 1d\nB : after x, 1d", 3, CodeTask},
		{"duplicate id", "gantt\nA : a, 2026-01-01, 1d\nB : a, 2026-01-02, 1d", 3, CodeTask},
		{"cycle", "gantt\nA : a, after b, 1d\nB : b, after a, 1d", 2, CodeTask},
		{"ends before start", "gantt\nA : 2026-01-10, 2026-01-01", 2, CodeDate},
		{"unsupported date token", "gantt\ndateFormat Do MMMM YYYY\nA : 2026-01-01, 1d", 2, CodeDate},
		{"dateFormat after task", "gantt\nA : 2026-01-01, 1d\ndateFormat DD.MM.YYYY", 3, CodeDate},
		{"bad excludes", "gantt\nexcludes święta\nA : 2026-01-01, 1d", 2, CodeDate},
		{"bad tick", "gantt\ntickInterval 2fortnight\nA : 2026-01-01, 1d", 2, CodeStatement},
		{"compact mode", "gantt\ndisplayMode compact\nA : 2026-01-01, 1d", 2, CodeUnsupported},
		{"typo in year", "gantt\nA : 2026-01-01, 202600d", 0, CodeDate},
	}
	for _, c := range cases {
		_, err := Parse(c.src)
		var se *syntax.Error
		if !errors.As(err, &se) {
			t.Errorf("%s: want a syntax error, got %v", c.name, err)
			continue
		}
		if se.Line != c.line || se.Code != c.code {
			t.Errorf("%s: got line %d code %q (%s), want line %d code %q", c.name, se.Line, se.Code, se.Message, c.line, c.code)
		}
	}
}

// REQ: GANTT-002 GANTT-004
func TestParseDates(t *testing.T) {
	d, err := Parse("gantt\nexcludes weekends, 2026-11-11\ninclusiveEndDates\n" +
		"A : a, 2026-11-06, 3d\nB : b, after a c, 1.5d\nC : c, 2026-11-02, 2026-11-04\nD : until e\nE : milestone, e, 2026-12-01, 0d\nF : 12h")
	if err != nil {
		t.Fatal(err)
	}
	at := func(s string) time.Time {
		v, err := time.Parse("2006-01-02 15:04", s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	want := map[string][2]time.Time{
		// Piątek + 3 dni robocze: piątek, poniedziałek, wtorek; koniec w środę 11.11 o północy.
		"A": {at("2026-11-06 00:00"), at("2026-11-11 00:00")},
		// Zaczyna się po A, ale 11.11 jest wyłączony: półtora dnia pracy od czwartku.
		"B": {at("2026-11-11 00:00"), at("2026-11-13 12:00")},
		"C": {at("2026-11-02 00:00"), at("2026-11-05 00:00")}, // koniec włącznie
		"D": {at("2026-11-05 00:00"), at("2026-12-01 00:00")},
		"E": {at("2026-12-01 00:00"), at("2026-12-01 00:00")},
		"F": {at("2026-12-01 00:00"), at("2026-12-01 12:00")},
	}
	for _, task := range d.Tasks {
		if w := want[task.Name]; !task.Start.Equal(w[0]) || !task.End.Equal(w[1]) {
			t.Errorf("%s: got %s – %s, want %s – %s", task.Name, task.Start.Format("01-02 15:04"), task.End.Format("01-02 15:04"), w[0].Format("01-02 15:04"), w[1].Format("01-02 15:04"))
		}
	}
	if !d.Tasks[4].Milestone || d.Tasks[1].ID != "b" {
		t.Errorf("tags or ids: %+v", d.Tasks)
	}
}

// REQ: GANTT-006
func TestStrftimeAndTicks(t *testing.T) {
	at := time.Date(2026, 3, 9, 7, 5, 0, 0, time.UTC)
	if got := strftime("%Y-%m-%d %e %H:%M %b %a %j %% %q", at); got != "2026-03-09 9 07:05 Mar Mon 068 % %q" {
		t.Errorf("strftime: %q", got)
	}
	lo, hi := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)
	if m := ticks(lo, hi, 1, "month"); len(m) != 3 || m[0].Month() != time.February || m[2].Month() != time.April {
		t.Errorf("month ticks: %v", m)
	}
	if w := ticks(lo, hi, 1, "week"); w[0].Weekday() != time.Monday || w[0].Day() != 19 {
		t.Errorf("week ticks start at %v", w[0])
	}
}
