package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/seriiserii825/go-libs/clipboard"
	"github.com/seriiserii825/go-libs/msg"
	"github.com/seriiserii825/go-libs/tools"

	"github.com/seriiserii825/go-scss/scss"
)

// Папка с файлами переменных проектов: projects/<имя папки проекта>.txt.
const projectsDir = "Documents/go/go-scss/projects"

func main() {
	toRem := flag.Bool("to-rem", false, "convert px to rem")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `Usage: go-scss [--to-rem]

Run from the project root (folder with package.json). Watches the
clipboard: CSS copied from Figma (text with ":") is converted and put
back into the clipboard:

  - letter-spacing px/%% -> em, line-height px/%% -> unitless
  - --to-rem: var(--x, 16px) and 16px -> 1.6rem
  - removes comments and font-feature-settings, overflow, ... lines
  - replaces values from ~/%s/<project>.txt
    (lines "find|replace", "find|delete" removes the line)
  - moves color before background, text-align/text-shadow before color

`, projectsDir)
		flag.PrintDefaults()
	}
	flag.Parse()

	if err := run(*toRem); err != nil {
		msg.Error(err.Error())
		os.Exit(1)
	}
}

func run(toRem bool) error {
	if err := tools.Require(tools.Clipboard(), tools.Notify); err != nil {
		return err
	}
	if _, err := os.Stat("package.json"); err != nil {
		return fmt.Errorf("file package.json not found")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	varsFile := filepath.Join(home, projectsDir, filepath.Base(cwd)+".txt")
	if _, err := scss.LoadVariables(varsFile); err != nil {
		return err
	}

	msg.Info("Watching clipboard, variables: " + varsFile)
	var last string
	for ; ; time.Sleep(500 * time.Millisecond) {
		text, err := clipboard.Read()
		if err != nil || text == last {
			continue
		}
		last = text
		if !strings.Contains(text, ":") {
			continue
		}
		// файл перечитывается каждый раз — правки видны без перезапуска
		vars, err := scss.LoadVariables(varsFile)
		if err != nil {
			msg.Error(err.Error())
			continue
		}
		out := scss.Process(text, vars, toRem)
		if err := clipboard.Write(out); err != nil {
			msg.Error(err.Error())
			continue
		}
		// clipboard.Read обрезает пробелы так же, как Write, — результат
		// не будет обработан повторно
		last = strings.TrimSpace(out)
		fmt.Println(out)
	}
}
