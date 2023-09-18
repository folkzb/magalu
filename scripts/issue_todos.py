# ATTENTION, run `grep -Rnw . -e TODO --include=\*.{go,sh,py} > todos.txt`
# on the root to generate updated todos.
# Usage: GH_TOKEN=<your_token> USERNAME=<username> python3 ./script/issue_todos.py <file path> # noqa: E501
# Get your Token following
# https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens#creating-a-fine-grained-personal-access-token # noqa: E501
# Use your github username

import os
import re
import sys
import http.client
import json
from dataclasses import dataclass, asdict, field

domain = "api.github.com"
path = "/profusion/magalu"


@dataclass
class GHIssue:
    title: str
    body: str
    assignee: str = ""
    milestone: int = None
    labels: [str] = field(default_factory=list)

    def to_json(self) -> str:
        return json.dumps(asdict(self))


ghConnection = http.client.HTTPSConnection(domain)


def open_issue(issue: GHIssue) -> tuple[http.client.HTTPResponse, bytes]:
    ghConnection.request(
        "POST",
        path + "/issues",
        issue.to_json(),
        {
            "Content-Type": "application/json",
            "Authorization": f"Bearer {os.environ.get('GH_TOKEN', '')}",
            "Accept": "application/vnd.github+json",
            "User-Agent": os.environ.get("USERNAME", ""),
        },
    )
    response = ghConnection.getresponse()
    body = response.read()
    return response, body


def strip_comment_content(comment: str) -> str:
    comment = comment.strip()
    # Comment start with //|# ... todo:? ...
    regex = r"(?:\/\/|#|\/\*).*TODO:?\s*(.*)$"
    matches = re.search(regex, comment, re.MULTILINE)
    return matches.group(1) if matches is not None else None


def get_gh_url(file: str, line: str) -> str:
    return f"https://github.com/{path}/blob/main{file.lstrip('. ')}#L{line}"


def get_description(file: str, line: str, stripped: str) -> str:
    linesStr = ""
    for line in line.split(","):
        linesStr = (
            linesStr
            + f"""
{file}:{line}
{get_gh_url(file, line)}

"""
        )

    return (
        f"""
There is a TODO comment present in file {file} at line {line}.
Please review the comment and fix the issue.
"""
        + linesStr
    )


with open(sys.argv[1], "r") as f:
    i = 0
    for todo in f.readlines():
        [file, line, comment] = todo.split(":", 2)
        stripped = strip_comment_content(comment)

        if stripped is None:
            print("Skipped: ", file, line, comment)
            continue

        issue = GHIssue(
            f"TODO: {stripped}",
            get_description(file, line, stripped),
            labels=["good first issue", "TODO"],
        )
        response, body = open_issue(issue)
        print(f"{i} -> [{response.getcode()}] {body}")
        i += 1

ghConnection.close()
