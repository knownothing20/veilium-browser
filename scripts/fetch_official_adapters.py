#!/usr/bin/env python3
"""Download and safely extract exact official adapter release assets."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import shutil
import stat
import sys
import tarfile
import urllib.request
import zipfile


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as source:
        while chunk := source.read(1024 * 1024):
            digest.update(chunk)
    return digest.hexdigest()


def load_manifest(path: Path) -> dict:
    data = json.loads(path.read_text(encoding="utf-8"))
    if data.get("schemaVersion") != 1 or not isinstance(data.get("releases"), list):
        raise ValueError("unsupported adapter release manifest")
    return data


def safe_member(name: str) -> PurePosixPath:
    candidate = PurePosixPath(name)
    if candidate.is_absolute() or ".." in candidate.parts or not candidate.parts:
        raise ValueError(f"unsafe archive member: {name}")
    return candidate


def download(url: str, destination: Path, expected_size: int, expected_sha256: str) -> None:
    request = urllib.request.Request(url, headers={"User-Agent": "veilium-official-adapter-fetcher"})
    digest = hashlib.sha256()
    size = 0
    with urllib.request.urlopen(request, timeout=180) as response, destination.open("wb") as output:
        while True:
            chunk = response.read(1024 * 1024)
            if not chunk:
                break
            output.write(chunk)
            digest.update(chunk)
            size += len(chunk)
    if size != expected_size or digest.hexdigest() != expected_sha256:
        destination.unlink(missing_ok=True)
        raise ValueError(f"downloaded asset failed pinned archive verification: {destination.name}")


def extract_zip(archive: Path, member_name: str, destination: Path) -> None:
    expected = safe_member(member_name).as_posix()
    with zipfile.ZipFile(archive) as source:
        infos = {safe_member(info.filename).as_posix(): info for info in source.infolist() if not info.is_dir()}
        info = infos.get(expected)
        if info is None:
            # Xray archives place xray at the root, while probing paths may retain extraction labels.
            basename_matches = [item for name, item in infos.items() if PurePosixPath(name).name == PurePosixPath(expected).name]
            if len(basename_matches) != 1:
                raise ValueError(f"expected executable {member_name} is missing from {archive.name}")
            info = basename_matches[0]
        mode = (info.external_attr >> 16) & 0o170000
        if mode == stat.S_IFLNK:
            raise ValueError("refusing symbolic-link executable from release archive")
        with source.open(info) as input_file, destination.open("xb") as output:
            shutil.copyfileobj(input_file, output)


def extract_tar(archive: Path, member_name: str, destination: Path) -> None:
    expected = safe_member(member_name).as_posix()
    with tarfile.open(archive, "r:gz") as source:
        members = {safe_member(member.name).as_posix(): member for member in source.getmembers() if member.isfile()}
        member = members.get(expected)
        if member is None:
            basename_matches = [item for name, item in members.items() if PurePosixPath(name).name == PurePosixPath(expected).name]
            if len(basename_matches) != 1:
                raise ValueError(f"expected executable {member_name} is missing from {archive.name}")
            member = basename_matches[0]
        if member.issym() or member.islnk():
            raise ValueError("refusing linked executable from release archive")
        input_file = source.extractfile(member)
        if input_file is None:
            raise ValueError("release executable could not be read")
        with input_file, destination.open("xb") as output:
            shutil.copyfileobj(input_file, output)


def write_github_output(path: str | None, key: str, value: str) -> None:
    if not path:
        return
    with open(path, "a", encoding="utf-8") as output:
        output.write(f"{key}={value}\n")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--manifest", default="internal/adapterrelease/releases.json")
    parser.add_argument("--platform", required=True, choices=["linux", "windows"])
    parser.add_argument("--arch", default="amd64", choices=["amd64"])
    parser.add_argument("--output-dir", required=True)
    parser.add_argument("--github-output", default=os.environ.get("GITHUB_OUTPUT"))
    args = parser.parse_args()

    manifest = load_manifest(Path(args.manifest))
    output_dir = Path(args.output_dir).resolve()
    output_dir.mkdir(parents=True, exist_ok=True)
    results: dict[str, str] = {}

    for release in manifest["releases"]:
        matches = [
            asset for asset in release["assets"]
            if asset["platform"] == args.platform and asset["arch"] == args.arch
        ]
        if len(matches) != 1:
            raise ValueError(f"expected one {release['kind']} asset for {args.platform}/{args.arch}")
        asset = matches[0]
        archive = output_dir / asset["name"]
        download(asset["url"], archive, asset["archiveSizeBytes"], asset["archiveSha256"])
        executable_name = "xray.exe" if release["kind"] == "xray" and args.platform == "windows" else "xray" if release["kind"] == "xray" else "sing-box.exe" if args.platform == "windows" else "sing-box"
        executable = output_dir / executable_name
        if asset["name"].endswith(".zip"):
            extract_zip(archive, asset["executablePath"], executable)
        elif asset["name"].endswith(".tar.gz"):
            extract_tar(archive, asset["executablePath"], executable)
        else:
            raise ValueError(f"unsupported adapter archive format: {asset['name']}")
        if executable.stat().st_size != asset["executableSizeBytes"] or sha256_file(executable) != asset["executableSha256"]:
            executable.unlink(missing_ok=True)
            raise ValueError(f"extracted {release['kind']} executable failed pinned verification")
        if args.platform != "windows":
            executable.chmod(0o700)
        key = "sing_box_path" if release["kind"] == "sing-box" else "xray_path"
        results[key] = str(executable)
        write_github_output(args.github_output, key, str(executable))

    print(json.dumps(results, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as error:  # noqa: BLE001 - CLI boundary
        print(f"official adapter fetch failed: {error}", file=sys.stderr)
        raise SystemExit(1)
