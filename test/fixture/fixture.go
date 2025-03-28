package fixture

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cucumber/messages-go/v10"
	"github.com/git-town/git-town/v8/src/config"
	"github.com/git-town/git-town/v8/src/stringslice"
	"github.com/git-town/git-town/v8/test/datatable"
	"github.com/git-town/git-town/v8/test/git"
	"github.com/git-town/git-town/v8/test/helpers"
	"github.com/git-town/git-town/v8/test/testruntime"
)

// Fixture is a complete Git environment for a Cucumber scenario.
type Fixture struct {
	// Dir defines the local folder in which this Fixture is stored.
	// This folder also acts as the HOME directory for tests using this Fixture.
	// It contains the global Git configuration to use in this test.
	Dir string

	// CoworkerRepo is the optional Git repository that is locally checked out at the coworker machine.
	CoworkerRepo *testruntime.TestRuntime `exhaustruct:"optional"`

	// DevRepo is the Git repository that is locally checked out at the developer machine.
	DevRepo testruntime.TestRuntime `exhaustruct:"optional"`

	// OriginRepo is the Git repository that simulates the origin repo (on GitHub).
	// If this value is nil, the current test setup has no origin.
	OriginRepo *testruntime.TestRuntime `exhaustruct:"optional"`

	// SubmoduleRepo is the Git repository that simulates an external repo used as a submodule.
	// If this value is nil, the current test setup uses no submodules.
	SubmoduleRepo *testruntime.TestRuntime `exhaustruct:"optional"`

	// UpstreamRepo is the optional Git repository that contains the upstream for this environment.
	UpstreamRepo *testruntime.TestRuntime `exhaustruct:"optional"`
}

// CloneFixture provides a Fixture instance in the given directory,
// containing a copy of the given Fixture.
func CloneFixture(original Fixture, dir string) (Fixture, error) {
	err := helpers.CopyDirectory(original.Dir, dir)
	if err != nil {
		return Fixture{}, fmt.Errorf("cannot clone Fixture %q to folder %q: %w", original.Dir, dir, err)
	}
	binDir := filepath.Join(dir, "bin")
	originDir := filepath.Join(dir, "origin")
	originRepo := testruntime.New(originDir, dir, "")
	developerDir := filepath.Join(dir, "developer")
	devRepo := testruntime.New(developerDir, dir, binDir)
	result := Fixture{
		Dir:        dir,
		DevRepo:    devRepo,
		OriginRepo: &originRepo,
	}
	// Since we copied the files from the memoized directory,
	// we have to set the "origin" remote to the copied origin repo here.
	err = result.DevRepo.Run("git", "remote", "remove", config.OriginRemote)
	if err != nil {
		return Fixture{}, fmt.Errorf("cannot remove remote: %w", err)
	}
	err = result.DevRepo.AddRemote(config.OriginRemote, result.originRepoPath())
	if err != nil {
		return Fixture{}, fmt.Errorf("cannot set remote: %w", err)
	}
	err = result.DevRepo.Fetch()
	if err != nil {
		return Fixture{}, fmt.Errorf("cannot fetch: %w", err)
	}
	// and connect the main branches again
	err = result.DevRepo.ConnectTrackingBranch("main")
	if err != nil {
		return Fixture{}, fmt.Errorf("cannot connect tracking branch: %w", err)
	}
	return result, err
}

// NewStandardFixture provides a Fixture in the given directory,
// fully populated as a standardized setup for scenarios.
//
// The origin repo has the initial branch checked out.
// Git repos cannot receive pushes of the currently checked out branch
// because that will change files in the current workspace.
// The tests don't use the initial branch.
func NewStandardFixture(dir string) (Fixture, error) {
	// create the folder
	// create the Fixture
	gitEnv := Fixture{Dir: dir}
	// create the origin repo
	err := os.MkdirAll(gitEnv.originRepoPath(), 0o744)
	if err != nil {
		return gitEnv, fmt.Errorf("cannot create directory %q: %w", gitEnv.originRepoPath(), err)
	}
	// initialize the repo in the folder
	originRepo, err := testruntime.Initialize(gitEnv.originRepoPath(), gitEnv.Dir, gitEnv.binPath())
	if err != nil {
		return gitEnv, err
	}
	err = originRepo.RunMany([][]string{
		{"git", "commit", "--allow-empty", "-m", "Initial commit"},
		{"git", "branch", "main", "initial"},
	})
	if err != nil {
		return gitEnv, fmt.Errorf("cannot initialize origin directory at %q: %w", gitEnv.originRepoPath(), err)
	}
	gitEnv.OriginRepo = &originRepo
	// clone the "developer" repo
	gitEnv.DevRepo, err = testruntime.Clone(originRepo.TestRunner, gitEnv.developerRepoPath())
	if err != nil {
		return gitEnv, fmt.Errorf("cannot clone developer repo %q from origin %q: %w", gitEnv.originRepoPath(), gitEnv.developerRepoPath(), err)
	}
	err = gitEnv.initializeWorkspace(&gitEnv.DevRepo)
	if err != nil {
		return gitEnv, fmt.Errorf("cannot create new standard Git environment: %w", err)
	}
	err = gitEnv.DevRepo.RemoveUnnecessaryFiles()
	if err != nil {
		return gitEnv, err
	}
	err = gitEnv.OriginRepo.RemoveUnnecessaryFiles()
	if err != nil {
		return gitEnv, err
	}
	return gitEnv, nil
}

// AddSubmodule adds a submodule repository.
func (env *Fixture) AddSubmoduleRepo() error {
	err := os.MkdirAll(env.submoduleRepoPath(), 0o744)
	if err != nil {
		return fmt.Errorf("cannot create directory %q: %w", env.submoduleRepoPath(), err)
	}
	submoduleRepo, err := testruntime.Initialize(env.submoduleRepoPath(), env.Dir, env.binPath())
	if err != nil {
		return err
	}
	err = submoduleRepo.RunMany([][]string{
		{"git", "config", "--global", "protocol.file.allow", "always"},
		{"git", "commit", "--allow-empty", "-m", "Initial commit"},
	})
	if err != nil {
		return fmt.Errorf("cannot initialize submodule directory at %q: %w", env.originRepoPath(), err)
	}
	env.SubmoduleRepo = &submoduleRepo
	return nil
}

// AddUpstream adds an upstream repository.
func (env *Fixture) AddUpstream() error {
	repo, err := testruntime.Clone(env.DevRepo.TestRunner, filepath.Join(env.Dir, "upstream"))
	if err != nil {
		return fmt.Errorf("cannot clone upstream: %w", err)
	}
	env.UpstreamRepo = &repo
	err = env.DevRepo.AddRemote("upstream", env.UpstreamRepo.WorkingDir)
	if err != nil {
		return fmt.Errorf("cannot set upstream remote: %w", err)
	}
	return nil
}

// AddCoworkerRepo adds a coworker repository.
func (env *Fixture) AddCoworkerRepo() error {
	coworkerRepo, err := testruntime.Clone(env.OriginRepo.TestRunner, env.coworkerRepoPath())
	if err != nil {
		return fmt.Errorf("cannot clone coworker: %w", err)
	}
	env.CoworkerRepo = &coworkerRepo
	return env.initializeWorkspace(env.CoworkerRepo)
}

// binPath provides the full path of the folder containing the test tools for this Fixture.
func (env *Fixture) binPath() string {
	return filepath.Join(env.Dir, "bin")
}

// Branches provides a tabular list of all branches in this Fixture.
func (env *Fixture) Branches() (datatable.DataTable, error) {
	result := datatable.DataTable{}
	result.AddRow("REPOSITORY", "BRANCHES")
	mainBranch := env.DevRepo.Config.MainBranch()
	localBranches, err := env.DevRepo.LocalBranchesMainFirst(mainBranch)
	if err != nil {
		return result, fmt.Errorf("cannot determine the developer repo branches of the Fixture: %w", err)
	}
	localBranches = stringslice.Remove(localBranches, "initial")
	localBranchesJoined := strings.Join(localBranches, ", ")
	if env.OriginRepo == nil {
		result.AddRow("local", localBranchesJoined)
		return result, nil
	}
	originBranches, err := env.OriginRepo.LocalBranchesMainFirst(mainBranch)
	if err != nil {
		return result, fmt.Errorf("cannot determine the origin repo branches of the Fixture: %w", err)
	}
	originBranches = stringslice.Remove(originBranches, "initial")
	originBranchesJoined := strings.Join(originBranches, ", ")
	if localBranchesJoined == originBranchesJoined {
		result.AddRow("local, origin", localBranchesJoined)
	} else {
		result.AddRow("local", localBranchesJoined)
		result.AddRow("origin", originBranchesJoined)
	}
	return result, nil
}

// CreateCommits creates the commits described by the given Gherkin table in this Git repository.
func (env *Fixture) CreateCommits(commits []git.Commit) error {
	for _, commit := range commits {
		var err error
		for _, location := range commit.Locations {
			switch location {
			case "coworker":
				err = env.CoworkerRepo.CreateCommit(commit)
			case "local":
				err = env.DevRepo.CreateCommit(commit)
			case "local, origin":
				err = env.DevRepo.CreateCommit(commit)
				if err != nil {
					return fmt.Errorf("cannot create local commit: %w", err)
				}
				err = env.DevRepo.PushBranchToRemote(commit.Branch, config.OriginRemote)
				if err != nil {
					return fmt.Errorf("cannot push branch %q after creating commit: %w", commit.Branch, err)
				}
			case "origin":
				err = env.OriginRepo.CreateCommit(commit)
			case "upstream":
				err = env.UpstreamRepo.CreateCommit(commit)
			default:
				return fmt.Errorf("unknown commit location %q", commit.Locations)
			}
		}
		if err != nil {
			return err
		}
	}
	// after setting up the commits, check out the "initial" branch in the origin repo so that we can git-push to it.
	if env.OriginRepo != nil {
		err := env.OriginRepo.CheckoutBranch("initial")
		if err != nil {
			return fmt.Errorf("cannot change origin repo back to initial: %w", err)
		}
	}
	return nil
}

// CreateOriginBranch creates a branch with the given name only in the origin directory.
func (env Fixture) CreateOriginBranch(name, parent string) error {
	err := env.OriginRepo.CreateBranch(name, parent)
	if err != nil {
		return fmt.Errorf("cannot create origin branch %q: %w", name, err)
	}
	return nil
}

// CreateTags creates tags from the given gherkin table.
func (env Fixture) CreateTags(table *messages.PickleStepArgument_PickleTable) error {
	columnNames := helpers.TableFields(table)
	if columnNames[0] != "NAME" && columnNames[1] != "LOCATION" {
		return fmt.Errorf("tag table must have columns NAME and LOCATION")
	}
	for _, row := range table.Rows[1:] {
		name := row.Cells[0].Value
		location := row.Cells[1].Value
		var err error
		switch location {
		case "local":
			err = env.DevRepo.CreateTag(name)
		case "origin":
			err = env.OriginRepo.CreateTag(name)
		default:
			err = fmt.Errorf("tag table LOCATION must be 'local' or 'origin'")
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// CommitTable provides a table for all commits in this Git environment containing only the given fields.
func (env Fixture) CommitTable(fields []string) (datatable.DataTable, error) {
	builder := datatable.NewCommitTableBuilder()
	localCommits, err := env.DevRepo.Commits(fields, "main")
	if err != nil {
		return datatable.DataTable{}, fmt.Errorf("cannot determine commits in the developer repo: %w", err)
	}
	builder.AddMany(localCommits, "local")
	if env.CoworkerRepo != nil {
		coworkerCommits, err := env.CoworkerRepo.Commits(fields, "main")
		if err != nil {
			return datatable.DataTable{}, fmt.Errorf("cannot determine commits in the coworker repo: %w", err)
		}
		builder.AddMany(coworkerCommits, "coworker")
	}
	if env.OriginRepo != nil {
		originCommits, err := env.OriginRepo.Commits(fields, "main")
		if err != nil {
			return datatable.DataTable{}, fmt.Errorf("cannot determine commits in the origin repo: %w", err)
		}
		builder.AddMany(originCommits, config.OriginRemote)
	}
	if env.UpstreamRepo != nil {
		upstreamCommits, err := env.UpstreamRepo.Commits(fields, "main")
		if err != nil {
			return datatable.DataTable{}, fmt.Errorf("cannot determine commits in the origin repo: %w", err)
		}
		builder.AddMany(upstreamCommits, "upstream")
	}
	return builder.Table(fields), nil
}

// TagTable provides a table for all tags in this Git environment.
func (env Fixture) TagTable() (datatable.DataTable, error) {
	builder := datatable.NewTagTableBuilder()
	localTags, err := env.DevRepo.Tags()
	if err != nil {
		return datatable.DataTable{}, err
	}
	builder.AddMany(localTags, "local")
	if env.OriginRepo != nil {
		originTags, err := env.OriginRepo.Tags()
		if err != nil {
			return datatable.DataTable{}, err
		}
		builder.AddMany(originTags, config.OriginRemote)
	}
	return builder.Table(), nil
}

func (env Fixture) initializeWorkspace(repo *testruntime.TestRuntime) error {
	err := repo.Config.SetMainBranch("main")
	if err != nil {
		return err
	}
	err = repo.Config.SetPerennialBranches([]string{})
	if err != nil {
		return err
	}
	return repo.RunMany([][]string{
		{"git", "checkout", "main"},
		// NOTE: the developer repos receives the initial branch from origin
		//       but we don't want it here because it isn't used in tests.
		{"git", "branch", "-d", "initial"},
	})
}

// coworkerRepoPath provides the full path to the Git repository with the given name.
func (env Fixture) coworkerRepoPath() string {
	return filepath.Join(env.Dir, "coworker")
}

// developerRepoPath provides the full path to the Git repository with the given name.
func (env Fixture) developerRepoPath() string {
	return filepath.Join(env.Dir, "developer")
}

// originRepoPath provides the full path to the Git repository with the given name.
func (env Fixture) originRepoPath() string {
	return filepath.Join(env.Dir, config.OriginRemote)
}

// submoduleRepoPath provides the full path to the Git repository with the given name.
func (env Fixture) submoduleRepoPath() string {
	return filepath.Join(env.Dir, "submodule")
}

// Remove deletes all files used by this Fixture from disk.
func (env Fixture) Remove() error {
	return os.RemoveAll(env.Dir)
}
