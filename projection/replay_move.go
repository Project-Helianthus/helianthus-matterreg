//go:build ignore

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: replay_move vectors out golden")
		os.Exit(2)
	}
	vectors, out, golden := os.Args[1], os.Args[2], os.Args[3]
	for _, rel := range []string{
		"current/gateway/matter",
		"current/gateway/matter-feedv1",
		"current/gateway/matter-libraryhost",
	} {
		if _, err := os.Stat(filepath.Join(vectors, rel)); err != nil {
			exit(err)
		}
	}
	runs := []struct {
		pkg string
		dir string
	}{
		{"./projection", filepath.Join(out, "matter")},
		{"./feedv1", filepath.Join(out, "matter-feedv1")},
		{"./libraryhost", filepath.Join(out, "matter-libraryhost")},
	}
	for _, run := range runs {
		if err := os.MkdirAll(run.dir, 0o755); err != nil {
			exit(err)
		}
		cmd := exec.Command("go", "test", "-tags", "replaymove", "-count=1", "-timeout", "5m", "-run", "^TestReplayMove$", run.pkg)
		cmd.Env = append(os.Environ(), "GOWORK=off", "GO111MODULE=on", "ORACLE_OUT="+run.dir, "HELIANTHUS_FEED_GOLDEN="+golden)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			exit(err)
		}
	}
}

func exit(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
