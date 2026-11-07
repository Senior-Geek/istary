//go:build !unix

package main

// Without flock (not unix), the save folder is not locked; see lock.go.

import "errors"

type saveLock struct{}

var errLockUnsupported = errors.New("this platform does not support locking the save folder")

func lockSaveFolder(dir string) (*saveLock, error) { return nil, errLockUnsupported }

func (l *saveLock) release() {}
