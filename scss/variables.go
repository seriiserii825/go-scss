package scss

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// LoadVariables читает файл переменных проекта (строки "искать|заменить").
// Нет файла — создаёт пустой и возвращает ошибку: его надо заполнить.
// Пустые строки из файла удаляются, строки без "|" пропускаются.
func LoadVariables(path string) ([]Variable, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("variables file not found, created empty %s — fill it", path)
	}
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("variables file %s is empty", path)
	}

	var kept []string
	var vars []Variable
	for line := range strings.SplitSeq(string(data), "\n") {
		if line == "" {
			continue
		}
		kept = append(kept, line)
		find, replace, ok := strings.Cut(line, "|")
		if !ok || find == "" {
			continue
		}
		replace, _, _ = strings.Cut(replace, "|")
		vars = append(vars, Variable{Find: find, Replace: replace})
	}

	if cleaned := strings.Join(kept, "\n") + "\n"; cleaned != string(data) {
		if err := os.WriteFile(path, []byte(cleaned), 0o644); err != nil {
			return nil, err
		}
	}
	return vars, nil
}
