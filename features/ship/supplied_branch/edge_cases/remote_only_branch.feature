@skipWindows
Feature: ship a branch that exists only on origin

  Background:
    Given the current branch is a feature branch "other"
    And a remote feature branch "feature"
    And the commits
      | BRANCH  | LOCATION | MESSAGE        | FILE NAME        |
      | feature | origin   | feature commit | conflicting_file |
    And an uncommitted file with name "conflicting_file" and content "conflicting content"
    When I run "git-town ship feature -m 'feature done'" and answer the prompts:
      | PROMPT                                        | ANSWER  |
      | Please specify the parent branch of 'feature' | [ENTER] |

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
      |         | git merge --no-edit main           |
      |         | git checkout main                  |
      | main    | git merge --squash feature         |
      |         | git commit -m "feature done"       |
      |         | git push                           |
      |         | git push origin :feature           |
      |         | git branch -D feature              |
      |         | git checkout other                 |
      | other   | git stash pop                      |
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

  Scenario: undo
    When I run "git-town undo"
    Then it runs the commands
      | BRANCH  | COMMAND                                       |
      | other   | git add -A                                    |
      |         | git stash                                     |
      |         | git checkout main                             |
      | main    | git branch feature {{ sha 'feature commit' }} |
      |         | git push -u origin feature                    |
      |         | git revert {{ sha 'feature done' }}           |
      |         | git push                                      |
      |         | git checkout feature                          |
      | feature | git checkout main                             |
      | main    | git checkout other                            |
      | other   | git stash pop                                 |
    And the current branch is now "other"
    And now these commits exist
      | BRANCH  | LOCATION      | MESSAGE               |
      | main    | local, origin | feature done          |
      |         |               | Revert "feature done" |
      | feature | local, origin | feature commit        |
    And the branches are now
      | REPOSITORY    | BRANCHES             |
      | local, origin | main, feature, other |
    And this branch hierarchy exists now
      | BRANCH  | PARENT |
      | feature | main   |
      | other   | main   |
