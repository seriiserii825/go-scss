// go-scss-host — Native Messaging host для расширения chrome-figma-copy.
//
// Chrome запускает его по запросу (connectNative), на один запрос, и он
// сразу завершается. Фонового процесса нет: без расширения ничего не работает
// и ничего не трогает буфер обмена.
//
// Протокол Chrome: на stdin/stdout сообщения вида
// [4 байта длины, little-endian][JSON].
//
// Запрос:  {"text": "...css...", "project": "ag-axel", "toRem": false}
// Ответ:   {"output": "...css..."} или {"error": "..."}
package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/seriiserii825/go-scss/scss"
)

// Та же папка, что и у go-scss: projects/<имя проекта>.txt.
const projectsDir = "Documents/go/go-scss/projects"

type request struct {
	Text    string `json:"text"`
	Project string `json:"project"`
	ToRem   bool   `json:"toRem"`
}

type response struct {
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

func main() {
	if err := serve(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "go-scss-host:", err)
		os.Exit(1)
	}
}

func serve(in io.Reader, out io.Writer) error {
	req, err := readMessage(in)
	if err != nil {
		return err
	}
	resp := handle(req)
	return writeMessage(out, resp)
}

func handle(req request) response {
	vars, err := loadProjectVars(req.Project)
	if err != nil {
		return response{Error: err.Error()}
	}
	return response{Output: scss.Process(req.Text, vars, req.ToRem)}
}

// Пустое имя проекта — конвертация без замен из файла переменных.
func loadProjectVars(project string) ([]scss.Variable, error) {
	project = strings.TrimSpace(project)
	if project == "" {
		return nil, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	// Имя файла не должно выходить за пределы папки проектов.
	if filepath.Base(project) != project || project == "." || project == ".." {
		return nil, fmt.Errorf("invalid project name %q", project)
	}
	return scss.LoadVariables(filepath.Join(home, projectsDir, project+".txt"))
}

func readMessage(r io.Reader) (request, error) {
	var length uint32
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return request{}, fmt.Errorf("read length: %w", err)
	}
	// Лимит на случай мусора на stdin; Chrome сам ограничивает сообщения 1 МБ.
	if length > 1<<20 {
		return request{}, fmt.Errorf("message too large: %d", length)
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return request{}, fmt.Errorf("read body: %w", err)
	}
	var req request
	if err := json.Unmarshal(buf, &req); err != nil {
		return request{}, fmt.Errorf("decode: %w", err)
	}
	return req, nil
}

func writeMessage(w io.Writer, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(len(data))); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
