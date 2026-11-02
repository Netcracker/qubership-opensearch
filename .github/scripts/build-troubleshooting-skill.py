#!/usr/bin/env python3
"""Build the troubleshoot-opensearch skill references from docs/.

The skill is installed without the rest of the repository, so every local
link must resolve inside the skill directory. This script copies the bundled
docs, trims what the skill does not need, and rewrites links:

* targets that are bundled (including GitHub URLs of this repository) become
  paths relative to the bundled file;
* ./installation.md#hwe becomes hwe.md (only that section is bundled);
* any other repository target becomes a GitHub URL.

It fails if a local link in the output still does not resolve.

Usage: build-troubleshooting-skill.py <output-references-dir>
"""

import posixpath
import re
import sys
from pathlib import Path

REPO_URL = "https://github.com/Netcracker/qubership-opensearch/blob/main/"
SCENARIOS = [
    "availability_zone_outage",
    "availability_zone_shutdown",
    "cpu_overload",
    "data_files_corrupted_on_primary_and_replica_shards",
    "data_files_corrupted_on_primary_shard",
    "data_files_corrupted_on_replica_shard",
    "disk_failure_on_all_nodes",
    "disk_failure_on_one_node",
    "disk_filled_on_all_nodes",
    "elected_master_is_crashed",
    "entire_index_corrupted",
    "io_limit",
    "memory_limit",
    "network_connection_failure",
    "new_master_can_not_be_elected",
    "primary_shard_is_down_during_user_request",
    "problem_during_replication",
    "translog_is_corrupted",
]

# repository source path -> path inside references/
BUNDLE = {
    "docs/public/troubleshooting.md": "troubleshooting.md",
    "docs/public/alerts.md": "alerts.md",
    "docs/public/indices-migration.md": "indices-migration.md",
    "docs/data/opensearch-tls-reinit.yaml": "opensearch-tls-reinit.yaml",
    **{f"docs/public/scenarios/{s}.md": f"scenarios/{s}.md" for s in SCENARIOS},
}
HWE_SOURCE = "docs/public/installation.md"
HWE_TARGET = "hwe.md"

LINK = re.compile(r"(\]\()([^)\s]+)((?:\s+\"[^\"]*\")?\))")


def strip_toc(text: str) -> str:
    """Drop the first <!-- TOC --> ... <!-- TOC --> block."""
    return re.sub(r"^<!-- TOC -->$.*?^<!-- TOC -->$\n+", "", text, count=1, flags=re.S | re.M)


def extract_hwe(text: str) -> str:
    """Keep only the '## HWE' section, up to the next '#' or '##' header."""
    match = re.search(r"^## HWE$.*?(?=^##? |\Z)", text, flags=re.S | re.M)
    if not match:
        sys.exit(f"'## HWE' section not found in {HWE_SOURCE}")
    return match.group(0).rstrip() + "\n"


def rewrite_links(text: str, source: str, target: str) -> str:
    def replace(match: re.Match) -> str:
        link = match.group(2)
        if link.lower().startswith(REPO_URL.lower()):
            link = "/" + link[len(REPO_URL):]
        elif re.match(r"^(?:[a-z][a-z0-9+.-]*:|#)", link, flags=re.I):
            return match.group(0)
        path, _, anchor = link.partition("#")
        if path.startswith("/"):
            repo_path = posixpath.normpath(path.lstrip("/"))
        else:
            repo_path = posixpath.normpath(posixpath.join(posixpath.dirname(source), path))

        if repo_path == HWE_SOURCE and anchor == "hwe":
            bundled, anchor = HWE_TARGET, ""
        else:
            bundled = BUNDLE.get(repo_path)

        if bundled:
            new = posixpath.relpath(bundled, posixpath.dirname(target) or ".")
        else:
            new = REPO_URL + repo_path
        if anchor:
            new += "#" + anchor
        return match.group(1) + new + match.group(3)

    return LINK.sub(replace, text)


def check_links(out: Path) -> list[str]:
    errors = []
    for file in sorted(out.rglob("*.md")):
        for number, line in enumerate(file.read_text(encoding="utf-8").splitlines(), 1):
            for match in LINK.finditer(line):
                link = match.group(2)
                if re.match(r"^(?:[a-z][a-z0-9+.-]*:|#)", link, flags=re.I):
                    continue
                path = link.partition("#")[0]
                if not (file.parent / path).is_file():
                    errors.append(f"{file.relative_to(out)}:{number}: unresolved link {link}")
    return errors


def main() -> None:
    if len(sys.argv) != 2:
        sys.exit(__doc__)
    repo = Path(__file__).resolve().parents[2]
    out = Path(sys.argv[1])

    outputs = {target: source for source, target in BUNDLE.items()}
    outputs[HWE_TARGET] = HWE_SOURCE
    for target, source in sorted(outputs.items()):
        text = (repo / source).read_text(encoding="utf-8").replace("\r\n", "\n")
        if target == "troubleshooting.md":
            text = strip_toc(text)
        elif target == HWE_TARGET:
            text = extract_hwe(text)
        if target.endswith(".md"):
            text = rewrite_links(text, source, target)
        destination = out / target
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_text(text, encoding="utf-8", newline="\n")

    errors = check_links(out)
    if errors:
        sys.exit("\n".join(errors))
    print(f"Built {len(outputs)} references into {out}")


if __name__ == "__main__":
    main()
