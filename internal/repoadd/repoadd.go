// Package repoadd backs the "add a repository" flow every channel offers (Telegram, CLI): clone the repo, then point a Linear project's `Repo:` directive at it.
package repoadd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ahmadAlMezaal/noctra/internal/github"
	"github.com/ahmadAlMezaal/noctra/internal/linear"
	"github.com/ahmadAlMezaal/noctra/internal/repo"
)

// Cloner is the repo.Resolver subset needed to clone on demand.
type Cloner interface {
	ResolveDirect(ctx context.Context, ref, branch string) (repo.Resolved, error)
}

// Projects is the linear.Client subset needed to read and rewrite routing directives.
type Projects interface {
	ListProjects(ctx context.Context) ([]linear.Project, error)
	UpdateProjectContent(ctx context.Context, projectID, content string) error
}

// Request is one add; an empty Branch means the repo's default, a nil Project means clone-only.
type Request struct {
	Ref     string
	Branch  string
	Project *linear.Project
}

type Result struct {
	OwnerRepo string
	Path      string
	Branch    string
	Project   string
}

// NormalizeRef validates a user-supplied reference, returning the ref to clone (URLs kept verbatim so non-GitHub hosts work) and its owner/name.
func NormalizeRef(input string) (ref, ownerRepo string, err error) {
	ref = strings.TrimSpace(input)
	if ref == "" {
		return "", "", errors.New("no repository given")
	}
	ownerRepo, err = github.ExtractOwnerRepo(ref)
	if err != nil {
		return "", "", err
	}
	return ref, ownerRepo, nil
}

// Add clones the repo, then writes the directive when a project was chosen. Cloning first means a directive never ends up pointing at a repo the host cannot reach; a Result with a Path plus an error means the clone succeeded and only the directive failed.
func Add(ctx context.Context, cloner Cloner, projects Projects, req Request) (Result, error) {
	ref, ownerRepo, err := NormalizeRef(req.Ref)
	if err != nil {
		return Result{}, err
	}

	resolved, err := cloner.ResolveDirect(ctx, ref, req.Branch)
	if err != nil {
		return Result{}, err
	}

	res := Result{OwnerRepo: ownerRepo, Path: resolved.Path, Branch: resolved.MainBranch}
	if req.Project == nil {
		return res, nil
	}
	if projects == nil {
		return res, errors.New("no Linear client is configured, so the routing directive was not written")
	}
	if req.Project.ID == "" {
		return res, fmt.Errorf("Linear project %q has no ID, so the routing directive was not written", req.Project.Name)
	}

	content := linear.UpsertRepoDirective(req.Project.Content, ref, req.Branch)
	if err := projects.UpdateProjectContent(ctx, req.Project.ID, content); err != nil {
		return res, fmt.Errorf("update Linear project %q: %w", req.Project.Name, err)
	}

	res.Project = req.Project.Name
	return res, nil
}

// ResolveProject finds the single project matching query, reporting the candidates when the answer is ambiguous.
func ResolveProject(ctx context.Context, projects Projects, query string) (*linear.Project, []linear.Project, error) {
	if projects == nil {
		return nil, nil, errors.New("no Linear client is configured")
	}

	all, err := projects.ListProjects(ctx)
	if err != nil {
		return nil, nil, err
	}

	matches := linear.MatchProjects(all, query)
	switch len(matches) {
	case 0:
		return nil, nil, fmt.Errorf("no Linear project matches %q", query)
	case 1:
		return &matches[0], nil, nil
	default:
		return nil, matches, nil
	}
}
