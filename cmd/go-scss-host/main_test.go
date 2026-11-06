package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"strings"
	"testing"
)

func TestServeRoundTrip(t *testing.T) {
	body, _ := json.Marshal(request{Text: "color: #fff;\nbackground: red;\n"})
	var in bytes.Buffer
	binary.Write(&in, binary.LittleEndian, uint32(len(body)))
	in.Write(body)

	var out bytes.Buffer
	if err := serve(&in, &out); err != nil {
		t.Fatal(err)
	}

	var length uint32
	binary.Read(&out, binary.LittleEndian, &length)
	var resp response
	if err := json.Unmarshal(out.Bytes()[:length], &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Error != "" || resp.Output == "" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestRejectsPathTraversal(t *testing.T) {
	if _, err := loadProjectVars("../secret"); err == nil {
		t.Fatal("expected error for traversal name")
	}
}

func TestListProjects(t *testing.T) {
	resp := handle(request{Action: "list"})
	if resp.Error != "" {
		t.Fatalf("list error: %s", resp.Error)
	}
	for _, p := range resp.Projects {
		if p == "" || strings.HasSuffix(p, ".txt") {
			t.Fatalf("bad project name %q", p)
		}
	}
}
