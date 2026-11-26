package sshconfig

import (
	"context"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	output := "user deploy\nhostname 203.0.113.7\nport 2222\nidentityfile ~/.ssh/missing\nidentityfile /keys/vps\nidentityfile /keys/other\n"
	got := parse(output, "/home/me", func(p string) bool { return p != filepath.Join("/home/me", ".ssh", "missing") })
	want := Host{HostName: "203.0.113.7", User: "deploy", Port: 2222, IdentityFile: "/keys/vps"}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	if !parse("hostname x\nproxyjump bastion\n", "", func(string) bool { return false }).Proxied {
		t.Error("proxy jump not detected")
	}
}

func TestLookupRefusesOptionLikeNames(t *testing.T) {
	if _, ok := Lookup(context.Background(), "-oProxyCommand=touch /tmp/x"); ok {
		t.Fatal("an option must never be passed to ssh")
	}
}
