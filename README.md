# githubmgr

This is a CLI tool to help you to manage your projects with GitHub.

* [Overview](#overview)
* [Config File](#config-file)
* [Option](#option)

## Overview

* `issue`: you can output task list for each assignees.

```text
$ ./githubmgr issue
# Issue & PR List for `phoenix-activity`
    task count: 7
    urgent: 10, 12

    ```
    - member-a       (3): 10, 13, 15
    - member-b       (2): 12, 15
    - member-c       (1): 9
    - (No Assignees) (2): 16, 17
    ```

@member-a, @member-b, @member-c
Please check the assigned issues.
```

## Config File

Please store the `config.json` file in the same directory as this tool. You can use any file name by specifying it with the command line option. Also, some properties in the config file can be specified on the command line.

```json:config.json
{
    "username": "default username",
    "repository": "default repository name",
    "access_token": "valid access token (if your repository is private, mandatory)",
    "message_to_assignee": "Please check the assigned issues.",
    "label_conditions": {
        "urgents": [
            "urgent"
        ],
        "pendings": [
            "pending"
        ]
    },
    "user_mappings": [
        {
            "github_name": "github name",
            "slack_name": "slack name"
        }
    ]
}
```

## Option

Several properties in the config file, such as `username` and `repository name`, can be specified on the command line. Please check help for details.