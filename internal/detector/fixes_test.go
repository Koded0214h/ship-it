package detector

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDetectRealisticManifests(t *testing.T) {
	cases := map[string]struct {
		files     map[string]string
		framework string
		db, work  bool
	}{
		"pip freeze casing": {map[string]string{"requirements.txt": "Django==5.0\n"}, "Django", false, false},
		"node pg":           {map[string]string{"package.json": `{"dependencies":{"express":"4","pg":"8"}}`}, "Express", true, false},
		"rails bullet":      {map[string]string{"Gemfile": "gem 'rails'\ngem 'bullet'\n"}, "Rails", false, false},
		"fastapi celery":    {map[string]string{"requirements.txt": "fastapi\ncelery\n"}, "FastAPI", false, true},
		"mysql url":         {map[string]string{"package.json": `{}`, ".env": "DATABASE_URL=mysql://x\n"}, "Node.js", false, false},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			for f, content := range c.files {
				writeFile(t, dir, f, content)
			}
			info, err := Detect(dir)
			if err != nil {
				t.Fatal(err)
			}
			if info.Framework != c.framework || info.HasDatabase != c.db || info.HasWorker != c.work {
				t.Errorf("got framework=%s db=%v worker=%v", info.Framework, info.HasDatabase, info.HasWorker)
			}
		})
	}
}

func TestProjectInfoJSONHasNoLocalPath(t *testing.T) {
	data, err := json.Marshal(&ProjectInfo{RootDir: "/Users/someone/secret-client"})
	if err != nil || strings.Contains(string(data), "secret-client") {
		t.Fatalf("local path would reach the AI provider: %s %v", data, err)
	}
}
