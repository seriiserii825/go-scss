// Package scss приводит CSS, скопированный из Figma, к виду для проекта:
// единицы типографики, px -> rem, замены из файла переменных, порядок свойств.
package scss

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Variable — строка файла переменных "искать|заменить". Replace == "delete"
// удаляет строку CSS целиком.
type Variable struct {
	Find    string
	Replace string
}

// Process прогоняет CSS через все шаги по порядку (как scssHandler в py-scss).
func Process(css string, vars []Variable, toRem bool) string {
	lines := splitLines(css)
	lines = convertTypography(lines)
	if toRem {
		lines = convertToRem(lines)
	}
	lines = removeComments(lines)
	lines = deleteLines(lines)
	lines = convertVariables(lines, vars)
	lines = sortLines(lines)
	lines = removeEmptyLines(lines)
	return strings.Join(lines, "")
}

// splitLines режет текст на строки с "\n" на конце у каждой, иначе
// последняя строка при переносе в sortLines склеилась бы со следующей.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.SplitAfter(s, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if last := lines[len(lines)-1]; !strings.HasSuffix(last, "\n") {
		lines[len(lines)-1] = last + "\n"
	}
	return lines
}

// formatNumber округляет до digits знаков без лишних нулей: 2 -> "2", -0 -> "0".
func formatNumber(f float64, digits int) string {
	p := math.Pow(10, float64(digits))
	s := strconv.FormatFloat(math.Round(f*p)/p, 'f', -1, 64)
	if s == "-0" {
		return "0"
	}
	return s
}

var (
	reVar           = regexp.MustCompile(`^var\([^,]+,\s*(.+)\)$`)
	reFontSize      = regexp.MustCompile(`font-size\s*:\s*([^;]+);`)
	reLetterSpacing = regexp.MustCompile(`(letter-spacing\s*:\s*)([^;]+);`)
	reLineHeight    = regexp.MustCompile(`(line-height\s*:\s*)([^;]+);`)
)

// unwrapVar: "var(--size-16, 16px)" -> "16px".
func unwrapVar(value string) string {
	value = strings.TrimSpace(value)
	if m := reVar.FindStringSubmatch(value); m != nil {
		return strings.TrimSpace(m[1])
	}
	return value
}

// parseNumber: ("16px", "px") -> 16, true; ("16rem", "px") -> 0, false.
func parseNumber(value, unit string) (float64, bool) {
	s, ok := strings.CutSuffix(unwrapVar(value), unit)
	if !ok || s == "" || strings.Trim(s, "-.0123456789") != "" || strings.LastIndex(s, "-") > 0 {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}

// fontSize — первый font-size в px (rem считается как px/10), 0 если нет.
func fontSize(lines []string) float64 {
	for _, line := range lines {
		m := reFontSize.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if px, ok := parseNumber(m[1], "px"); ok {
			return px
		}
		if rem, ok := parseNumber(m[1], "rem"); ok {
			return rem * 10
		}
	}
	return 0
}

// relative переводит px (относительно font-size) или % в доли.
func relative(value string, size float64) (float64, bool) {
	if px, ok := parseNumber(value, "px"); ok && size != 0 {
		return px / size, true
	}
	if percent, ok := parseNumber(value, "%"); ok {
		return percent / 100, true
	}
	return 0, false
}

// convertTypography: letter-spacing px/% -> em, line-height px/% -> без единиц.
func convertTypography(lines []string) []string {
	size := fontSize(lines)
	for i, line := range lines {
		line = reLetterSpacing.ReplaceAllStringFunc(line, func(s string) string {
			m := reLetterSpacing.FindStringSubmatch(s)
			em, ok := relative(m[2], size)
			if !ok {
				return s
			}
			n := formatNumber(em, 3)
			if n != "0" {
				n += "em"
			}
			return m[1] + n + ";"
		})
		line = reLineHeight.ReplaceAllStringFunc(line, func(s string) string {
			m := reLineHeight.FindStringSubmatch(s)
			ratio, ok := relative(m[2], size)
			if !ok {
				return s
			}
			return m[1] + formatNumber(ratio, 3) + ";"
		})
		lines[i] = line
	}
	return lines
}

var (
	reVarPx = regexp.MustCompile(`var\([^,]+,\s*([\d.]+)px\)`)
	rePx    = regexp.MustCompile(`([\d.]+)px`)
)

// В строках с этими подстроками px не трогаем.
var remIgnored = []string{"border: 1px", "linear-gradient", "&"}

// pxToRem заменяет совпадение re (группа 1 — число px) на N/10 rem.
func pxToRem(re *regexp.Regexp, line string) string {
	return re.ReplaceAllStringFunc(line, func(s string) string {
		px, err := strconv.ParseFloat(re.FindStringSubmatch(s)[1], 64)
		if err != nil {
			return s
		}
		return formatNumber(px/10, 2) + "rem"
	})
}

// convertToRem: var(--x, 16px) -> 1.6rem, затем оставшиеся px -> rem.
func convertToRem(lines []string) []string {
	for i, line := range lines {
		line = pxToRem(reVarPx, line)
		if strings.Contains(line, "px") && !containsAny(line, remIgnored) {
			line = pxToRem(rePx, line)
		}
		lines[i] = line
	}
	return lines
}

// removeComments обрезает строку по "/*" вместе с пробелами перед ним:
// "width: 10px; /* 28.8px */" -> "width: 10px;".
func removeComments(lines []string) []string {
	for i, line := range lines {
		if before, _, ok := strings.Cut(line, "/*"); ok {
			lines[i] = strings.TrimRight(before, " \t") + "\n"
		}
	}
	return lines
}

var deletedProps = []string{
	"font-feature-settings",
	"text-overflow",
	"overflow",
	"font-kerning",
	"leading-trim",
	"text-edge",
}

// deleteLines убирает строки с ненужными свойствами.
func deleteLines(lines []string) []string {
	for i, line := range lines {
		if containsAny(line, deletedProps) {
			lines[i] = ""
		}
	}
	return lines
}

// convertVariables заменяет Find на Replace (в нижнем регистре) без учёта
// регистра; Replace == "delete" удаляет строку.
func convertVariables(lines []string, vars []Variable) []string {
	for i := range lines {
		for _, v := range vars {
			if !strings.Contains(strings.ToLower(lines[i]), strings.ToLower(v.Find)) {
				continue
			}
			if v.Replace == "delete" {
				lines[i] = ""
			}
			re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(v.Find))
			lines[i] = re.ReplaceAllLiteralString(lines[i], strings.ToLower(v.Replace))
		}
	}
	return lines
}

// sortLines ставит color перед background, text-align и text-shadow — перед color.
func sortLines(lines []string) []string {
	lines = moveLine(lines, "color:", "background:")
	lines = moveLine(lines, "text-align:", "color:")
	lines = moveLine(lines, "text-shadow:", "color:")
	return lines
}

// moveLine переносит первую строку с prop перед первой строкой с before,
// а если такой нет — в конец.
func moveLine(lines []string, prop, before string) []string {
	from := indexContains(lines, prop)
	if from < 0 {
		return lines
	}
	line := lines[from]
	lines = append(lines[:from], lines[from+1:]...)
	to := indexContains(lines, before)
	if to < 0 {
		to = len(lines)
	}
	return append(lines[:to], append([]string{line}, lines[to:]...)...)
}

func removeEmptyLines(lines []string) []string {
	var out []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}

func indexContains(lines []string, sub string) int {
	for i, line := range lines {
		if strings.Contains(line, sub) {
			return i
		}
	}
	return -1
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
