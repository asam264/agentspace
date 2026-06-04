package cmd

import "errors"

var errNotGitRepo = errors.New("not a git repository (run this inside a git repo)")
