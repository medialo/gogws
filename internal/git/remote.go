package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func AddRemotes(ctx context.Context, targetPath string, remotes []*Remote, run func(context.Context, *exec.Cmd) error) error {
	if len(remotes) == 0 {
		return nil
	}

	if run == nil {
		run = defaultRunner
	}

	// 1. Récupérer les remotes existants (ex: "origin")
	var outBuf bytes.Buffer
	listCmd := exec.CommandContext(ctx, "git", "remote")
	listCmd.Dir = targetPath
	listCmd.Stdout = &outBuf

	if err := run(ctx, listCmd); err != nil {
		return fmt.Errorf("failed to list remotes: %w", err)
	}

	// 2. Stocker les résultats dans une map
	existingRemotes := make(map[string]bool)
	for _, r := range strings.Split(outBuf.String(), "\n") {
		r = strings.TrimSpace(r)
		if r != "" {
			existingRemotes[r] = true
		}
	}

	// 3. Ajouter uniquement les remotes manquants
	for _, remote := range remotes {
		if existingRemotes[remote.Name] {
			continue // Le remote existe déjà, on l'ignore
		}

		addCmd := exec.CommandContext(ctx, "git", "remote", "add", remote.Name, remote.URL)
		addCmd.Dir = targetPath
		if err := run(ctx, addCmd); err != nil {
			return fmt.Errorf("failed to add remote %s: %w", remote.Name, err)
		}

		// Mise à jour de la map pour éviter les doublons au sein de la liste 'remotes'
		existingRemotes[remote.Name] = true
	}

	return nil
}
