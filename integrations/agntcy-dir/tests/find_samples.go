// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"fmt"
	"os"
	"path/filepath"
)

func FindFilePairs(rootDir, buildConfigName, expectedModelName string) ([]string, error) {
	directories := make(map[string]int)

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			dir := filepath.Dir(path)
			if info.Name() == buildConfigName || info.Name() == expectedModelName {
				directories[dir]++
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking the path %q: %v", rootDir, err)
	}

	dirs := make([]string, 0)
	for dir, count := range directories {
		if count >= 2 {
			dirs = append(dirs, dir)
		}
	}

	return dirs, nil
}
