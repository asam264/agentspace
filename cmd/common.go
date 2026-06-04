package cmd

import (
	"github.com/asam264/agentspace/internal/git"
	"github.com/asam264/agentspace/internal/workspace"
)

// resolvePaths finds the repo root from the current directory and returns the
// agentspace paths. It does not require .agentspace to exist.
func resolvePaths() (workspace.Paths, error) {
	if !git.IsRepo("") {
		return workspace.Paths{}, errNotGitRepo
	}
	root, err := git.MainRoot("")
	if err != nil {
		return workspace.Paths{}, err
	}
	return workspace.NewPaths(root), nil
}

// requireInit resolves paths and ensures agentspace has been initialized.
func requireInit() (workspace.Paths, *workspace.Config, error) {
	p, err := resolvePaths()
	if err != nil {
		return p, nil, err
	}
	if !p.Initialized() {
		return p, nil, workspace.ErrNotInitialized
	}
	cfg, err := p.LoadConfig()
	if err != nil {
		return p, nil, err
	}
	return p, cfg, nil
}
