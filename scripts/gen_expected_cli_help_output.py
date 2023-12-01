#!/usr/bin/env python3

from typing import Iterator

import argparse
import difflib
import json
import logging
import os.path
import shutil
import subprocess
import sys

logging.basicConfig(format="%(levelname).3s %(message)s")
logger = logging.getLogger(__name__)


def iter_tree(
    children: list,
    parent_path: list[str],
) -> Iterator[list[str]]:
    for child in children:
        assert isinstance(child, dict)
        path = parent_path + [child["name"]]
        yield path

        grand_children = child.get("children")
        if not grand_children:
            continue

        for child_path in iter_tree(grand_children, path):
            yield child_path


def gen_cli_paths(cli: str) -> Iterator[list[str]]:
    args = [cli, "dump-tree", "-o", "json"]
    with subprocess.Popen(args, stdout=subprocess.PIPE, encoding="utf-8") as p:
        tree = json.load(p.stdout)
        assert isinstance(tree, list)
        yield [cli]
        for p in iter_tree(tree, [cli]):
            yield p


def gen_output(cmd: list[str]) -> str:
    logger.debug("running %s", cmd)
    return subprocess.run(
        cmd,
        encoding="utf-8",
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        env={},  # we don't want to contaminate it with any configs/flags set
        timeout=5,
    ).stdout


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
    parser.add_argument(
        "output_directory",
        type=str,
        help="the root folder where to write help output",
    )
    parser.add_argument(
        "-v",
        "--verbose",
        action="count",
        default=int(os.environ.get("VERBOSE", "0")),
    )
    args = parser.parse_args()

    logger.setLevel(logging.WARNING - (args.verbose * 10))

    root_dir: str = os.path.abspath(args.output_directory)
    logger.info("removing output-dir: %s", root_dir)
    try:
        shutil.rmtree(root_dir)
    except FileNotFoundError:
        pass

    for path in gen_cli_paths(args.cli):
        logger.info("processing: %s", path)
        help_output = gen_help_output(path)
        h_flag = gen_output_h_flag(path)
        p = " ".join(path[1:])
        check_output(p, help_output, h_flag)

        out_dir = os.path.join(root_dir, *path[1:])
        os.makedirs(out_dir, exist_ok=True)
        filepath = os.path.join(out_dir, "help.txt")
        with open(filepath, "w", encoding="utf-8") as f:
            f.write(help_output)
            logger.debug("wrote %s", filepath)
