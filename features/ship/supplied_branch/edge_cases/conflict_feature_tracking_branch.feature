Feature: handle conflicts between the supplied feature branch and its tracking branch

  Background:
    Given the feature branches "feature" and "other"
    And the commits
      | BRANCH  | LOCATION | MESSAGE                   | FILE NAME        | FILE CONTENT   |
      | feature | local    | conflicting local commit  | conflicting_file | local content  |
      |         | origin   | conflicting origin commit | conflicting_file | origin content |
    And the current branch is "other"
    And an uncommitted file
    And I run "git-town ship feature -m 'feature done'"

  Scenario: result
    Then it runs the commands
      | BRANCH  | COMMAND                            |
      | other   | git fetch --prune --tags           |
      |         | git add -A                         |
      |         | git stash                          |
      |         | git checkout main                  |
      | main    | git rebase origin/main             |
      |         | git checkout feature               |
      | feature | git merge --no-edit origin/feature |
    And it prints the error:
      """
      To abort, run "git-town abort".
      To continue after having resolved conflicts, run "git-town continue".
      """
    And the current branch is now "feature"
    And the uncommitted file is stashed
    And a merge is now in progress

  Scenario: abort
    When I run "git-town abort"
    Then it runs the commands
      | BRANCH  | COMMAND            |
      | feature | git merge --abort  |
      |         | git checkout main  |
      | main    | git checkout other |
      | other   | git stash pop      |
    And the current branch is now "other"
    And the uncommitted file still exists
    And no merge is in progress
    And now the initial commits exist
    And the initial branch hierarchy exists

  Scenario: resolve and continue
    When I resolve the conflict in "conflicting_file"
    And I run "git-town continue"
    Then it runs the commands
      | BRANCH  | COMMAND                      |
      | feature | git commit --no-edit         |
      |         | git merge --no-edit main     |
      |         | git checkout main            |
      | main    | git merge --squash feature   |
      |         | git commit -m "feature done" |
      |         | git push                     |
      |         | git push origin :feature     |
      |         | git branch -D feature        |
      |         | git checkout other           |
      | other   | git stash pop                |
    And the current branch is now "other"
    And the uncommitted file still exists
    And the branches are now
      | REPOSITORY    | BRANCHES    |
      | local, origin | main, other |
    And now these commits exist
      | BRANCH | LOCATION      | MESSAGE      |
      | main   | local, origin | feature done |
    And this branch hierarchy exists now
      | BRANCH | PARENT |
      | other  | main   |

  Scenario: resolve, commit, and continue
    When I resolve the conflict in "conflicting_file"
    And I run "git commit --no-edit"
    And I run "git-town continue"
    Then it runs the commands
      | BRANCH  | COMMAND                      |
      | feature | git merge --no-edit main     |
      |         | git checkout main            |
      | main    | git merge --squash feature   |
      |         | git commit -m "feature done" |
      |         | git push                     |
      |         | git push origin :feature     |
      |         | git branch -D feature        |
      |         | git checkout other           |
      | other   | git stash pop                |
    And the current branch is now "other"
    And the uncommitted file still exists

  Scenario: resolve, continue, and undo
    When I resolve the conflict in "conflicting_file"
    And I run "git-town continue"
    And I run "git-town undo"
    Then it runs the commands
      | BRANCH  | COMMAND                                                                                   |
      | other   | git add -A                                                                                |
      |         | git stash                                                                                 |
      |         | git checkout main                                                                         |
      | main    | git branch feature {{ sha 'Merge remote-tracking branch 'origin/feature' into feature' }} |
      |         | git push -u origin feature                                                                |
      |         | git revert {{ sha 'feature done' }}                                                       |
      |         | git push                                                                                  |
      |         | git checkout feature                                                                      |
      | feature | git checkout main                                                                         |
      | main    | git checkout other                                                                        |
      | other   | git stash pop                                                                             |
    And the current branch is now "other"
    And now these commits exist
      | BRANCH  | LOCATION      | MESSAGE                                                    |
      | main    | local, origin | feature done                                               |
      |         |               | Revert "feature done"                                      |
      | feature | local, origin | conflicting local commit                                   |
      |         |               | conflicting origin commit                                  |
      |         |               | Merge remote-tracking branch 'origin/feature' into feature |
    And the initial branches and hierarchy exist
