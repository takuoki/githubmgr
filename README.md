# githubmgr

This is a CLI tool to help you to manage your projects with GitHub.

## Overview

### issue

you can output task list for each assignees.

```text
$ ./githubmgr issue -p
# Issue & PR List for `test-user/test-repository`
    task count: 7
    urgent: 10, 12

    *Assingee List*
    ```
    - member-a       (3): 10, 13, 15
    - member-b       (2): 12, 15
    - member-c       (1): 9
    - (No Assignees) (2): 16, 17
    ```

    *Priority List*
    ```
    - urgent
      - 10: member-a
      - 12: member-b
    - major
      - 9 : member-c
      - 15: member-a, member-b
    - minor
      - 16: (No Assignees)
    - pending
      - 13: member-a
    - (No Priority Labels)
      - 17: (No Assignees)
    ```

@member-a, @member-b, @member-c
Please check the assigned issues.
```

### label

you can set some labels at one time with json settings.

```json
{
    "labels": [
        {"name": "urgent", "color": "ff3000", "desc": "Priority is urgent"},
        {"name": "critical", "color": "ff9305", "desc": "Priority is critical"},
        {"name": "major", "color": "fbca04", "desc": "Priority is major"},
        {"name": "minor", "color": "5ecc36", "desc": "Priority is minor"},
        {"name": "pending", "color": "2d86ee", "desc": "Priority is pending"},
        {"name": "bug", "color": "e03000", "desc": "Something isn't working"},
        {"name": "duplicate", "color": "cfd3d7", "desc": "This issue or pull request already exists"}
    ],
    "replace": [
        {"from": "wontfix", "to": "pending"}
    ],
    "ignore": [
        "question",
        "area/*"
    ]
}
```

* `labels`: create or update these labels
* `replace`: if `from` label exists and is attached to some issues or PRs, replace to `to` label
* `ignore`: if these label exists, do nothing
* if some other labels exists in your repository, these labels are deleted automatically.
  but if these labels are attached to some issues or PRs, this tool return error

```txt
$ ./githubmgr label
# Label settings for `test-user/test-repository`
  * label settings
    `urgent`: create (color="ff3000", desc="Priority is urgent")
    `critical`: create (color="ff9305", desc="Priority is critical")
    `major`: create (color="fbca04", desc="Priority is major")
    `minor`: create (color="5ecc36", desc="Priority is minor")
    `pending`: create (color="2d86ee", desc="Priority is pending")
    `bug`: update (color="d73a4a" -> "2d86ee", desc="Something isn't working")
    `duplicate`: update (color="cfd3d7" -> "cfd3d7", desc="This issue or pull request already exists")

  * replace labels
    `wontfix`: replace to `pending` (issues=10, 13)

  * ignore labels
    `question`
    `area/abc`
    `area/xyz`

  * delete labels
    `good first issue`
    `help wanted`
    `invalid`
    `enhancement` (issues=14)

  There is a label attached to issues in the delete labels.
  Please dettatch it from issues or write a label definition.
```

## Config File

Please store the `config.json` file in the same directory as this tool. You can use any file name by specifying it with the command line option. Also, some properties in the config file can be specified on the command line.

```json:config.json
{
    "username": "default_username",
    "repository": "default_repository_name",
    "access_token": "valid_access_token",
    "message_to_assignee": "Please check the assigned issues.",
    "label_rule" : {
        "priority": [
            {"label_name": "urgent", "level":"High"},
            {"label_name": "critical", "level":"High"},
            {"label_name": "major", "level":"Middle"},
            {"label_name": "minor", "level":"Middle"},
            {"label_name": "pending", "level":"Low"}
        ],
        "other": [
            {"label_name": "bug", "level":"High"},
            {"label_name": "wontfix", "level":"Low"}
        ]
    },
    "user_mappings": [
        {
            "github_name": "github_name",
            "slack_name": "slack_name"
        }
    ]
}
```

## Option

Several properties in the config file, such as `username` and `repository name`, can be specified on the command line. Please check help for details.
