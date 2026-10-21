package scss

import (
	"os"
	"path/filepath"
	"testing"
)

var testVars = []Variable{
	{"Instrument Sans", "delete"},
	{"#F2F2F2", "var(--text)"},
	{"var(--Red, #E0373C)", "var(--accent)"},
	{"var(--Blue, #4C7CAC)", "var(--contrast)"},
}

// Ожидаемые значения получены прогоном тех же входных данных через py-scss.
func TestProcess(t *testing.T) {
	figma := `color: var(--Blue, #4C7CAC);
font-family: Instrument Sans;
font-size: var(--size-16, 16px);
font-style: normal;
font-weight: 600;
line-height: 24px; /* 150% */
letter-spacing: -0.32px;
text-align: center;
background: var(--Red, #E0373C);
overflow: hidden;
text-overflow: ellipsis;`

	box := `display: flex;
width: 345px;
padding: var(--spacing-12, 12px) 20px;
border: 1px solid #F2F2F2;
gap: 10px;
font-size: 18px;
line-height: 120%;
letter-spacing: 2%;
background: #F2F2F2;`

	shadow := `text-shadow: 0 1px 2px #000;
font-size: 1.4rem;
letter-spacing: 0px;
line-height: normal;
border-radius: 8.5px;`

	tests := []struct {
		name  string
		in    string
		toRem bool
		want  string
	}{
		{"figma", figma, false, `font-size: var(--size-16, 16px);
font-style: normal;
font-weight: 600;
line-height: 1.5;
letter-spacing: -0.02em;
text-align: center;
color: var(--contrast);
background: var(--accent);
`},
		{"figma rem", figma, true, `font-size: 1.6rem;
font-style: normal;
font-weight: 600;
line-height: 1.5;
letter-spacing: -0.02em;
text-align: center;
color: var(--contrast);
background: var(--accent);
`},
		{"box", box, false, `display: flex;
width: 345px;
padding: var(--spacing-12, 12px) 20px;
border: 1px solid var(--text);
gap: 10px;
font-size: 18px;
line-height: 1.2;
letter-spacing: 0.02em;
background: var(--text);
`},
		{"box rem", box, true, `display: flex;
width: 34.5rem;
padding: 1.2rem 2rem;
border: 1px solid var(--text);
gap: 1rem;
font-size: 1.8rem;
line-height: 1.2;
letter-spacing: 0.02em;
background: var(--text);
`},
		{"shadow", shadow, false, `font-size: 1.4rem;
letter-spacing: 0;
line-height: normal;
border-radius: 8.5px;
text-shadow: 0 1px 2px #000;
`},
		{"shadow rem", shadow, true, `font-size: 1.4rem;
letter-spacing: 0;
line-height: normal;
border-radius: 0.85rem;
text-shadow: 0 0.1rem 0.2rem #000;
`},
		{"empty", "", true, ""},
	}
	for _, tt := range tests {
		if got := Process(tt.in, testVars, tt.toRem); got != tt.want {
			t.Errorf("%s:\ngot:\n%s\nwant:\n%s", tt.name, got, tt.want)
		}
	}
}

func TestMoveLineLastWithoutNewline(t *testing.T) {
	// последняя строка без "\n" не должна склеиться со следующей
	got := Process("background: red;\ncolor: blue;", nil, false)
	if want := "color: blue;\nbackground: red;\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestParseNumber(t *testing.T) {
	tests := []struct {
		in, unit string
		want     float64
		ok       bool
	}{
		{"16px", "px", 16, true},
		{" var(--size-16, 16px) ", "px", 16, true},
		{"-0.32px", "px", -0.32, true},
		{"120%", "%", 120, true},
		{"1.4rem", "px", 0, false},
		{"normal", "px", 0, false},
		{"1.2.3px", "px", 0, false},
		{"1-2px", "px", 0, false},
	}
	for _, tt := range tests {
		got, ok := parseNumber(tt.in, tt.unit)
		if got != tt.want || ok != tt.ok {
			t.Errorf("parseNumber(%q, %q) = %v, %v", tt.in, tt.unit, got, ok)
		}
	}
}

func TestLoadVariables(t *testing.T) {
	dir := t.TempDir()

	missing := filepath.Join(dir, "missing.txt")
	if _, err := LoadVariables(missing); err == nil {
		t.Error("missing file: want error")
	}
	if _, err := os.Stat(missing); err != nil {
		t.Errorf("missing file was not created: %v", err)
	}
	if _, err := LoadVariables(missing); err == nil {
		t.Error("empty file: want error")
	}

	path := filepath.Join(dir, "vars.txt")
	os.WriteFile(path, []byte("Instrument Sans|delete\n\n#F2F2F2|var(--text)\n123\n\n"), 0o644)
	vars, err := LoadVariables(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []Variable{{"Instrument Sans", "delete"}, {"#F2F2F2", "var(--text)"}}
	if len(vars) != len(want) || vars[0] != want[0] || vars[1] != want[1] {
		t.Errorf("vars = %v, want %v", vars, want)
	}
	data, _ := os.ReadFile(path)
	if got := string(data); got != "Instrument Sans|delete\n#F2F2F2|var(--text)\n123\n" {
		t.Errorf("file not cleaned: %q", got)
	}
}
