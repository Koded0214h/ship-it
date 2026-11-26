package sshkey

import (
	"os"
	"testing"
)

func TestGenerateNeverOverwrites(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))
	first, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(first.PrivatePath)
	if _, err := Generate(); err == nil {
		t.Fatal("expected an error for an existing key")
	}
	after, _ := os.ReadFile(first.PrivatePath)
	if string(before) != string(after) {
		t.Fatal("existing key was replaced")
	}
}
