Feature: does not sync branches that exist only on remotes

  Background:
    Given a feature branch "mine"
    And a remote feature branch "other"
    And the commits
      | BRANCH | LOCATION      | MESSAGE         |
      | main   | origin        | main commit     |
      | mine   | local, origin | my commit       |
      | other  | origin        | coworker commit |
    And the current branch is "main"
    When I run "git-town sync --all"

  Scenario: result
    Then it runs the commands
      | BRANCH | COMMAND                         |
      | main   | git fetch --prune --tags        |
      |        | git rebase origin/main          |
      |        | git checkout mine               |
      | mine   | git merge --no-edit origin/mine |
      |        | git merge --no-edit main        |
      |        | git push                        |
      |        | git checkout main               |
      | main   | git push --tags                 |
    And the current branch is still "main"
    And all branches are now synchronized
    And now these commits exist
      | BRANCH | LOCATION      | MESSAGE                       |
      | main   | local, origin | main commit                   |
      | mine   | local, origin | my commit                     |
      |        |               | main commit                   |
      |        |               | Merge branch 'main' into mine |
      | other  | origin        | coworker commit               |
