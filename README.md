# githubmgr

This is a CLI tool to help you to manage your projects with GitHub.

## Overview

### issue

you can output task list for each assignees.

```text
$ ./githubmgr issue
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
