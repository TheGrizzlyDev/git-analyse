package gitfs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/TheGrizzlyDev/git-analyse/settings"
)

var casCache map[string]struct{}
var casMu sync.RWMutex

func init() {
	casCache = make(map[string]struct{})
}

type gitfs struct {
}

func New() *gitfs {
	return &gitfs{}
}

func (g *gitfs) Ls(ctx context.Context, rev string) ([]*FileRevision, error) {
	var out bytes.Buffer

	execCmd := exec.CommandContext(ctx, "git", "ls-tree", "-r", "--full-name", rev)

	execCmd.Stdout = &out

	if err := execCmd.Run(); err != nil {
		return nil, err
	}

	revs := []*FileRevision{}
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		fields := strings.Fields(line)
		hash, path := fields[2], fields[3]
		if fileMode, err := strconv.ParseInt(fields[0][2:], 8, 32); err != nil {
			return nil, err
		} else {
			revs = append(revs, &FileRevision{
				Hash: hash,
				Path: path,
				Rev:  rev,
				Mode: os.FileMode(fileMode),
			})
		}
	}

	return revs, nil
}

type FileRevision struct {
	Hash string
	Path string
	Rev  string
	Mode os.FileMode
	mu   sync.Mutex
}

func (f *FileRevision) initIfNeeded(ctx context.Context) {
	casMu.Lock()
	defer casMu.Unlock()
	if _, ok := casCache[f.Hash]; ok {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, err := os.Stat(f.casPath()); err == nil {
		return
	} else if errors.Is(err, os.ErrNotExist) {
		var out bytes.Buffer
		execCmd := exec.CommandContext(ctx, "git", "show", fmt.Sprintf("%s:%s", f.Rev, f.Path))
		execCmd.Stdout = &out
		execCmd.Run()
		os.WriteFile(f.casPath(), out.Bytes(), f.Mode.Perm())
		casCache[f.Hash] = struct{}{}
	} else {
		panic(err)
	}
}

func (f *FileRevision) casPath() string {
	return path.Join(settings.CasPath, f.Hash)
}

func (f *FileRevision) Show(ctx context.Context) ([]byte, error) {
	f.initIfNeeded(ctx)
	return os.ReadFile(f.casPath())
}

func (f *FileRevision) Link(ctx context.Context, dest string) error {
	f.initIfNeeded(ctx)
	return os.Symlink(f.casPath(), dest)
}
