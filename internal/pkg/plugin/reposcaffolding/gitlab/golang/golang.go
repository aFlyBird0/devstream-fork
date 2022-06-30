package golang

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"strings"

	rs "github.com/devstream-io/devstream/internal/pkg/plugin/common/reposcaffolding"
	"github.com/devstream-io/devstream/pkg/util/gitlab"
	"github.com/devstream-io/devstream/pkg/util/log"
)

const (
	DefaultWorkPath      = ".gitlab-repo-scaffolding-golang"
	TransitBranch        = "init-with-devstream"
	DefaultMainBranch    = "main"
	DefaultCommitMessage = "initialized by DevStream"
)

type Config struct {
	AppName   string
	ImageRepo string
	Repo      Repo
}

type Repo struct {
	Name  string
	Owner string
}

func pushToRemote(repoPath string, opts *rs.Options) error {
	// create a GitLab client
	c, err := gitlab.NewClient(gitlab.WithBaseURL(opts.BaseURL))
	if err != nil {
		return err
	}

	var retErr error
	createProjOpts := &gitlab.CreateProjectOptions{
		Name:       opts.Repo,
		Branch:     opts.Branch,
		Namespace:  opts.Org,
		Visibility: opts.Visibility,
	}

	// create the project
	if err := c.CreateProject(createProjOpts); err != nil {
		log.Errorf("Failed to create repo: %s.", err)
		return err
	}

	log.Infof("The repo %s has been created.", opts.Repo)

	defer func() {
		if retErr == nil {
			return
		}
		log.Errorf("Failed to push to remote: %s.", retErr)
		// need to clean the repo created when retErr != nil
		if err := c.DeleteProject(opts.PathWithNamespace); err != nil {
			log.Errorf("Failed to delete the repo %s: %s.", opts.PathWithNamespace, err)
		}
	}()

	if retErr = walkLocalRepoPath(repoPath, opts, c); retErr != nil {
		log.Debugf("Failed to walk local repo-path: %s.", retErr)
		return retErr
	}

	return nil
}

func walkLocalRepoPath(repoPath string, opts *rs.Options, c *gitlab.Client) error {
	mainBranch := getMainBranchName(opts)

	var files = make(map[string][]byte)

	if err := filepath.Walk(repoPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			log.Debugf("Walk error: %s.", err)
			return err
		}

		if info.IsDir() {
			log.Debugf("Found dir: %s.", path)
			return nil
		}

		log.Debugf("Found file: %s.", path)

		content, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		gitlabPath := strings.Join(strings.Split(path, "/")[2:], "/")
		files[gitlabPath] = content
		return nil
	}); err != nil {
		return err
	}

	return c.CommitMultipleFiles(opts.PathWithNamespace, mainBranch, DefaultCommitMessage, files)
}

func getMainBranchName(opts *rs.Options) string {
	if opts.Branch == "" {
		return DefaultMainBranch
	}
	return opts.Branch
}

func buildState(opts *rs.Options) map[string]interface{} {
	res := make(map[string]interface{})
	res["owner"] = opts.Owner
	res["org"] = opts.Org
	res["repoName"] = opts.Repo

	outputs := make(map[string]interface{})
	outputs["owner"] = opts.Owner
	outputs["org"] = opts.Org
	outputs["repo"] = opts.Repo

	var gitlabURL string
	if opts.BaseURL != "" {
		gitlabURL = opts.BaseURL
	} else {
		gitlabURL = gitlab.DefaultGitlabHost
	}

	if opts.Org != "" {
		outputs["repoURL"] = fmt.Sprintf("%s/%s/%s.git", gitlabURL, opts.Org, opts.Repo)
	} else {
		outputs["repoURL"] = fmt.Sprintf("%s/%s/%s.git", gitlabURL, opts.Owner, opts.Repo)
	}
	res["outputs"] = outputs

	return res
}
