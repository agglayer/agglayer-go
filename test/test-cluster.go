package test

import (
	"os"
	"os/exec"
	"path/filepath"
)

type testCluster struct {
	path            string
	originalWorkDir string
}

func newTestCluster(path string) (*testCluster, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	if path == "" {
		parentDir := filepath.Dir(workDir)
		if err := os.Chdir(parentDir); err != nil {
			return nil, err
		}
	}

	return &testCluster{
		path:            path,
		originalWorkDir: workDir,
	}, nil
}

func (t *testCluster) start() ([]byte, error) {
	return exec.Command("make", "run-docker").CombinedOutput()
}

func (t *testCluster) stop() ([]byte, error) {
	return exec.Command("make", "stop-docker").CombinedOutput()
}

func (t *testCluster) reset() error {
	return os.Chdir(t.originalWorkDir)
}
