#!/usr/bin/env python3
"""Exercise official local adapters with a headless Chromium request."""

from __future__ import annotations

import argparse
import http.server
import json
import os
from pathlib import Path
import signal
import socket
import subprocess
import tempfile
import threading
import time

TOKEN = "VEILIUM_ADAPTER_CHROMIUM_SMOKE_OK"


class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self) -> None:  # noqa: N802
        body = f"<html><body>{TOKEN}</body></html>".encode()
        self.send_response(200)
        self.send_header("Content-Type", "text/html")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, _format: str, *_args: object) -> None:
        return


def free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as listener:
        listener.bind(("127.0.0.1", 0))
        return int(listener.getsockname()[1])


def wait_socks(port: int, process: subprocess.Popen[bytes]) -> None:
    deadline = time.time() + 12
    last_error: Exception | None = None
    while time.time() < deadline:
        if process.poll() is not None:
            raise RuntimeError(f"adapter exited before readiness with {process.returncode}")
        try:
            with socket.create_connection(("127.0.0.1", port), timeout=0.5) as connection:
                connection.sendall(b"\x05\x01\x00")
                if connection.recv(2) == b"\x05\x00":
                    return
        except OSError as error:
            last_error = error
        time.sleep(0.1)
    raise RuntimeError(f"adapter SOCKS5 listener was not ready: {last_error}")


def chrome_fetch(chromium: str, proxy_port: int, target_url: str, user_data: Path) -> str:
    command = [
        chromium,
        "--headless=new",
        "--no-sandbox",
        "--disable-gpu",
        "--disable-background-networking",
        "--disable-component-update",
        "--disable-default-apps",
        "--disable-extensions",
        "--disable-sync",
        "--metrics-recording-only",
        "--no-first-run",
        f"--user-data-dir={user_data}",
        f"--proxy-server=socks5://127.0.0.1:{proxy_port}",
        "--proxy-bypass-list=<-loopback>",
        "--dump-dom",
        target_url,
    ]
    result = subprocess.run(command, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=30, check=False)
    return result.stdout.decode("utf-8", errors="replace")


def adapter_config(kind: str, port: int) -> tuple[list[str], dict]:
    if kind == "xray":
        return ["run", "-config"], {
            "log": {"loglevel": "warning", "access": "none"},
            "inbounds": [{
                "tag": "veilium-smoke",
                "listen": "127.0.0.1",
                "port": port,
                "protocol": "socks",
                "settings": {"auth": "noauth", "udp": True, "ip": "127.0.0.1"},
            }],
            "outbounds": [{"tag": "direct", "protocol": "freedom"}],
            "routing": {"rules": [{"type": "field", "inboundTag": ["veilium-smoke"], "outboundTag": "direct"}]},
        }
    return ["run", "-c"], {
        "log": {"disabled": True},
        "inbounds": [{"type": "socks", "tag": "veilium-smoke", "listen": "127.0.0.1", "listen_port": port}],
        "outbounds": [{"type": "direct", "tag": "direct"}],
        "route": {"final": "direct"},
    }


def run_one(kind: str, executable: str, chromium: str, target_url: str) -> None:
    with tempfile.TemporaryDirectory(prefix=f"veilium-{kind}-smoke-") as raw_directory:
        directory = Path(raw_directory)
        port = free_port()
        args, config = adapter_config(kind, port)
        config_path = directory / "config.json"
        config_path.write_text(json.dumps(config), encoding="utf-8")
        os.chmod(config_path, 0o600)
        log_path = directory / "adapter.log"
        with log_path.open("wb") as log:
            process = subprocess.Popen([executable, *args, str(config_path)], stdout=log, stderr=log, start_new_session=True)
            try:
                wait_socks(port, process)
                output = chrome_fetch(chromium, port, target_url, directory / "chrome-data")
                if TOKEN not in output:
                    raise RuntimeError(f"Chromium did not render the smoke token through {kind}; output: {output[-2000:]}")
            finally:
                if process.poll() is None:
                    try:
                        os.killpg(process.pid, signal.SIGTERM)
                    except ProcessLookupError:
                        pass
                    try:
                        process.wait(timeout=5)
                    except subprocess.TimeoutExpired:
                        os.killpg(process.pid, signal.SIGKILL)
                        process.wait(timeout=5)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--xray", required=True)
    parser.add_argument("--sing-box", required=True)
    parser.add_argument("--chromium", required=True)
    args = parser.parse_args()

    server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        target_url = f"http://127.0.0.1:{server.server_port}/"
        # Confirm Chromium does not bypass an unavailable SOCKS proxy for loopback.
        failed_output = chrome_fetch(args.chromium, free_port(), target_url, Path(tempfile.mkdtemp(prefix="veilium-negative-smoke-")))
        if TOKEN in failed_output:
            raise RuntimeError("Chromium bypassed the unavailable SOCKS proxy during the negative control")
        run_one("xray", args.xray, args.chromium, target_url)
        run_one("sing-box", args.sing_box, args.chromium, target_url)
    finally:
        server.shutdown()
        server.server_close()
    print("official Xray and sing-box successfully proxied a headless Chromium request")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
