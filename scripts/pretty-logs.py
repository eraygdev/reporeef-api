#!/usr/bin/env python3
"""Pretty-printer for RepoReef backend logs (JSON lines).

Usage:
    docker compose logs --no-log-prefix -f backend | python3 scripts/pretty-logs.py
"""

import json
import sys

RESET = "\033[0m"


def c(code, text):
    return f"\033[{code}m{text}{RESET}"


DIM = lambda s: c("90", s)
RED = lambda s: c("31", s)
GREEN = lambda s: c("32", s)
YELLOW = lambda s: c("33", s)
BLUE = lambda s: c("34", s)
MAGENTA = lambda s: c("35", s)
CYAN = lambda s: c("36", s)
WHITE = lambda s: c("37", s)

LEVELS = {
    "DEBUG": DIM("DEBUG"),
    "INFO": GREEN("INFO "),
    "WARN": YELLOW("WARN "),
    "ERROR": RED("ERROR"),
}


def status_colored(code):
    if code >= 500:
        return RED(str(code))
    if code >= 400:
        return YELLOW(str(code))
    if code >= 200:
        return GREEN(str(code))
    return str(code)


for line in sys.stdin:
    line = line.strip()
    if not line:
        continue

    try:
        d = json.loads(line)
    except json.JSONDecodeError:
        print(line, flush=True)
        continue

    t = d.get("time", "?")[11:19]
    lvl = LEVELS.get(d.get("level", ""), d.get("level", "?"))

    if d.get("method"):
        method = CYAN(d["method"].ljust(7))
        path = WHITE(d["path"].ljust(38))
        status_raw = d.get("status", 0)
        status = status_colored(int(status_raw)).rjust(
            3 + 9
        )  # ANSI kodları ~9 karakter
        duration = DIM(f"{d.get('duration_ms', 0)}ms".rjust(8))
        out = f"{DIM(t)} {lvl} {method} {path} {status} {duration}"
        if d.get("ip"):
            out += f" {DIM('ip=' + str(d['ip']))}"
        if d.get("user_id"):
            out += f" {MAGENTA('u=' + str(d['user_id']))}"
        if d.get("repo"):
            out += f" {BLUE(d['repo'])}"
        if d.get("err"):
            out += f" {RED(d['err'])}"
    else:
        msg = WHITE(d.get("msg", "?"))
        out = f"{DIM(t)} {lvl} {msg}"
        if d.get("repo"):
            out += f" {BLUE(d['repo'])}"
        if d.get("addr"):
            out += f" {DIM(d['addr'])}"
        if d.get("err"):
            out += f" {RED(d['err'])}"

    print(out, flush=True)
