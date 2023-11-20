package test

import (
	"os"
	"os/exec"
	"path/filepath"
)

type testCluster struct {
	path        string
	origWorkDir string
}

func newTestCluster(path string) (*testCluster, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	if path == "" {
		path = filepath.Dir(workDir)
	}

	if err := os.Chdir(path); err != nil {
		return nil, err
	}

	return &testCluster{
		path:        path,
		origWorkDir: workDir,
	}, nil
}

func (t *testCluster) start() ([]byte, error) {
	return exec.Command("make", "run-docker").CombinedOutput()
}

func (t *testCluster) stop() ([]byte, error) {
	return exec.Command("make", "stop-docker").CombinedOutput()
}

func (t *testCluster) reset() error {
	return os.Chdir(t.origWorkDir)
}
