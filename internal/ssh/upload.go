package ssh

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxUploadFileSize = 30 * 1024 * 1024

// Directories that are never source code, skipped at any depth.
var skipDirs = map[string]bool{
	".git": true, ".ship": true, "node_modules": true, "__pycache__": true,
	".venv": true, "venv": true, ".idea": true, ".next": true,
}

// IsSecretFile reports whether a file name looks like it holds credentials.
// Such files are never uploaded.
func IsSecretFile(name string) bool {
	n := strings.ToLower(name)
	for _, suffix := range []string{".example", ".sample", ".template", ".dist"} {
		if strings.HasSuffix(n, suffix) {
			return false
		}
	}
	if n == ".env" || strings.HasPrefix(n, ".env.") || strings.HasSuffix(n, ".env") {
		return true
	}
	switch n {
	case "id_rsa", "id_ed25519", "id_ecdsa", "id_dsa", "ship_ed25519", ".npmrc", ".pypirc", ".netrc":
		return true
	}
	switch filepath.Ext(n) {
	case ".pem", ".key", ".p12", ".pfx":
		return true
	}
	return false
}

// buildArchive tars localDir. skipped describes anything left out that the user
// might have expected to be there.
func buildArchive(ctx context.Context, localDir string) (*bytes.Buffer, []string, error) {
	root, err := filepath.EvalSymlinks(localDir)
	if err != nil {
		return nil, nil, err
	}
	has := func(name string) bool {
		_, err := os.Stat(filepath.Join(root, name))
		return err == nil
	}
	// Top-level build output.
	topLevel := map[string]bool{"dist": true, "build": true}
	if has("Cargo.toml") || has("pom.xml") {
		topLevel["target"] = true
	}
	if has("composer.json") || has("Gemfile") {
		topLevel["vendor"] = true
	}

	var buf bytes.Buffer
	var skipped []string
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		slashed := filepath.ToSlash(rel)

		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(path, "pyvenv.cfg")); err == nil {
				return filepath.SkipDir
			}
			if topLevel[slashed] {
				skipped = append(skipped, slashed+"/ (build output)")
				return filepath.SkipDir
			}
			return nil // tar recreates directories from file paths
		}
		if skipDirs[info.Name()] {
			return nil // e.g. a .git file in a worktree
		}
		if IsSecretFile(info.Name()) {
			skipped = append(skipped, slashed+" (looks like a secret)")
			return nil
		}

		link := ""
		if info.Mode()&os.ModeSymlink != 0 {
			target, readErr := os.Readlink(path)
			resolved, evalErr := filepath.EvalSymlinks(path)
			if readErr != nil || evalErr != nil || filepath.IsAbs(target) ||
				!strings.HasPrefix(resolved, root+string(filepath.Separator)) || IsSecretFile(filepath.Base(resolved)) {
				skipped = append(skipped, slashed+" (symlink that is broken, absolute, or leaves the project)")
				return nil
			}
			link = filepath.ToSlash(target)
		} else if !info.Mode().IsRegular() {
			return nil
		} else if info.Size() > maxUploadFileSize {
			skipped = append(skipped, slashed+" (larger than 30 MB)")
			return nil
		}

		hdr, err := tar.FileInfoHeader(info, link)
		if err != nil {
			return err
		}
		hdr.Name = slashed
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if link != "" {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
	if err != nil {
		return nil, nil, fmt.Errorf("building source archive: %w", err)
	}
	if err := tw.Close(); err != nil {
		return nil, nil, fmt.Errorf("building source archive: %w", err)
	}
	if err := gw.Close(); err != nil {
		return nil, nil, fmt.Errorf("building source archive: %w", err)
	}
	return &buf, skipped, nil
}

// UploadDir tars localDir and extracts it into remoteDir on the server. Files
// already on the server that are not in the archive are left alone.
func (c *Client) UploadDir(ctx context.Context, localDir, remoteDir string) ([]string, error) {
	archive, skipped, err := buildArchive(ctx, localDir)
	if err != nil {
		return nil, err
	}
	var errOut bytes.Buffer
	cmd := fmt.Sprintf("mkdir -p %s && tar xzf - -C %s", Quote(remoteDir), Quote(remoteDir))
	if err := c.run(ctx, cmd, archive, io.Discard, &errOut); err != nil {
		return skipped, fmt.Errorf("extract source archive: %w\n%s", err, errOut.String())
	}
	return skipped, nil
}
