#!/usr/bin/env python3

from typing import Iterator

import argparse
import difflib
import json
import subprocess
import sys


def iter_tree(
    children: list[dict],
    parent_path: list[str],
) -> Iterator[list[str]]:
    for child in children:
        path = parent_path + [child["name"]]
        yield path

        grand_children = child.get("children")
        if not grand_children:
            continue

        for child_path in sorted(iter_tree(grand_children, path)):
            yield child_path


def gen_cli_paths(cli: str) -> Iterator[list[str]]:
    args = [cli, "dump-tree", "-o", "json"]
    with subprocess.Popen(args, stdout=subprocess.PIPE, encoding="utf-8") as p:
        tree = json.load(p.stdout)
        assert isinstance(tree, list)
        return iter_tree(tree, [cli])


def gen_output(cmd: list[str]) -> str:
    return subprocess.run(cmd, capture_output=True, encoding="utf-8").stdout


def gen_help_output(path: list[str]) -> str:
    cmd = [path[0], "help"] + path[1:]
    return gen_output(cmd)


def gen_output_h_flag(path: list[str]) -> str:
    cmd = path + ["-h"]
    return gen_output(cmd)


def check_output(p: str, help_output: str, h_flag: str) -> None:
    if help_output == h_flag:
        return

    name_help = json.dumps(f"help {p}")
    name_flag = json.dumps(f"{p} -h")
    diff = difflib.unified_diff(
        help_output.splitlines(keepends=True),
        h_flag.splitlines(keepends=True),
        name_help,
        name_flag,
    )
    sys.stderr.writelines(diff)
    raise ValueError(f"{name_help} and {name_flag} differ")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Generate expected CLI help output",
    )
    parser.add_argument(
        "cli",
        type=str,
        help="the binary to use during executions",
    )
    args = parser.parse_args()

    for path in gen_cli_paths(args.cli):
        help_output = gen_help_output(path)
        h_flag = gen_output_h_flag(path)
        p = " ".join(path[1:])
        check_output(p, help_output, h_flag)

        print(f"# {p}")
        print(help_output)
