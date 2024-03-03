#!/usr/bin/python3

import os, shutil, subprocess, sys

BASE_DIR = "/home/cipherboy/GitHub/cipherboy/openbao"
BIN_DIR = BASE_DIR + "/bin"
CMD_DIR = BASE_DIR + "/command"
WORK_DIR = "/home/cipherboy/tmp/3"

BINS = ['agent', 'cli', 'proxy', 'server']
TARGETS = ['dev', 'dev-agent', 'dev-cli', 'dev-proxy', 'dev-server']

EXCLUDE_PATHS = [
    "commands_agent.go",
    "commands_cli.go",
    "commands_nonwindows.go",
    "commands_windows.go",
    "commands_proxy.go",
    "commands_server.go",
    "commands_unified.go",
    "server_profile.go",
    "server_noprofile.go"
]

TARGET_WEIGHTS = {
    'agent': 2,
    'cli': 4,
    'proxy': 2,
    'server': 1,
}

shutil.rmtree(WORK_DIR, ignore_errors=True)
os.mkdir(WORK_DIR)

def do_build(target_dir=None, targets=TARGETS):
    if target_dir:
        os.mkdir(target_dir)

    for target in targets:
        shutil.rmtree(BIN_DIR, ignore_errors=True)
        if subprocess.call(['make', target], cwd=BASE_DIR) != 0:
            return False, target

        if target_dir:
            shutil.copyfile(BIN_DIR + "/bao", target_dir + "/" + target)

    return True, None

def apply_targets(path, targets):
    agent = targets['agent']
    cli = targets['cli']
    proxy = targets['proxy']
    server = targets['server']

    agent_prefix = '!agent' if not agent else ''
    cli_prefix = '!cli' if not cli else ''
    proxy_prefix = '!proxy' if not proxy else ''
    server_prefix = '!server' if not server else ''

    clauses = []
    for clause in [agent_prefix, cli_prefix, proxy_prefix, server_prefix]:
        if clause:
            clauses.append(clause)

    build_target = ""
    if not agent or not cli or not proxy or not server:
        build_target = f"//go:build " + " && ".join(clauses)

    with open(path, 'r') as f:
        lines = f.readlines()

    found_build = None
    for index, line in enumerate(lines):
        if line.startswith('//go:build'):
            found_build = index
            break

    if found_build is None:
        for index, line in enumerate(lines):
            if line.startswith('package command'):
                found_build = index
                break
        if found_build is None:
            raise Exception(f"no package statement in {path}")
        if build_target:
            lines = lines[0:found_build] + [build_target + "\n", "\n"] + lines[found_build:]
    else:
        lines[found_build] = build_target + "\n"

    with open(path, 'w') as f:
        for line in lines:
            f.write(line)
            if not line.endswith("\n"):
                f.write("\n")

def get_score_sizes(path):
    score = 0
    sizes = {}
    for file in os.listdir(path):
        if not file.startswith('dev'):
            continue

        size = os.stat(f"{path}/{file}").st_size
        sizes[file] = size
        for name, weight in TARGET_WEIGHTS.items():
            if name in file:
                score += size * weight

    return score, sizes

def list_to_targets(required):
    result = {}
    for target in BINS:
        result[target] = target in required
    return result

def get_file_target_requirements(file_dir, path):
    required_targets = []

    # assumption: we build now
    for exclude_target in BINS:
        # start fresh
        subprocess.call(["git", "checkout", "HEAD", "--", path])

        # remove only one target
        target_states = {
            exclude_target: False,
        }

        for other_states in BINS:
            if other_states == exclude_target:
                continue
            target_states[other_states] = True
        apply_targets(path, target_states)

        # check if the desired binary builds without this file
        build_dir = file_dir + f"/exclude-{exclude_target}"
        ok, _ = do_build(build_dir, ["dev-" + exclude_target])
        if not ok:
            required_targets.append(exclude_target)

    # now validate all targets coexist peacefully
    subprocess.call(["git", "checkout", "HEAD", "--", path])

    final_states = list_to_targets(required_targets)
    apply_targets(path, final_states)

    build_dir = file_dir + f"/validation"
    ok, failed = do_build(build_dir)
    if not ok:
        raise Exception(f"build independence assertion failed for {path}: failed to build {failed}, with build targets {final_states}, when succeeded individually")

    return final_states

def get_all_file_target_requirements(generation_dir, paths):
    constraints = {}
    for path in paths:
        filename = os.path.basename(path)
        file_dir = generation_dir + f"/reduce-{filename}"
        os.mkdir(file_dir)

        reqs = get_file_target_requirements(file_dir, path)
        constraints[path] = reqs

        shutil.rmtree(file_dir)

    return constraints

def main():
    paths = []
    for path in os.listdir(CMD_DIR):
        if path in EXCLUDE_PATHS:
            continue

        if path.endswith(".go") and not path.endswith("_test.go"):
            paths.append(f"{CMD_DIR}/{path}")

    print(f"{len(paths)} files to minimize...", file=sys.stderr)

    initial_dir = WORK_DIR + "/initial"
    ok, failed = do_build(initial_dir)
    if not ok:
        print(f"initial build failed on `make {failed}`", file=sys.stderr)
        sys.exit(1)

    initial_score, initial_sizes = get_score_sizes(initial_dir)
    print(f"initial score: {initial_score}\n\tsizes: {initial_sizes}", file=sys.stderr)

    last_states = None
    last_score = None
    lats_sizes = None
    changed = True
    generation = 3
    while changed:
        generation_dir = WORK_DIR + f"/generation-{generation}"
        os.mkdir(generation_dir)

        file_states = get_all_file_target_requirements(generation_dir, paths)

        if last_states is None:
            last_states = file_states
        else:
            changed = False
            for path, states in file_states.items():
                if states != last_states[path]:
                    changed = True
                    break

        # Add everything together and see what happens.
        subprocess.call(["git", "add", "-A"])
        subprocess.call(["git", "commit", "-m", f"generation {generation} reductions"])

        do_build(generation_dir + "/final")
        last_score, last_sizes = get_score_sizes(initial_dir)
        print(f"generation {generation} score: {last_score}\n\tsizes: {last_sizes}", file=sys.stderr)

        generation += 1

    print(f"initial score: {initial_score}\n\tsizes: {initial_sizes}", file=sys.stderr)
    print(f"last score: {last_score}\n\tsizes: {last_sizes}", file=sys.stderr)
    print(f"delta: {(initial_score-last_score)/initial_score*100}% reduction", file=sys.stderr)
    for target, last_size in last_sizes.items():
        initial_size = initial_sizes[target]
        print(f"\tdelta @ {target}: {(initial_size-last_size)/initial_size*100}% reduction", file=sys.stderr)

if __name__ == "__main__":
    main()
