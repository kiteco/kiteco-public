import os
import sys
import signal
import shlex
import shutil
import pkgutil
import argparse
import tempfile
import threading
import traceback
import subprocess

# Log file locations
EXPLORE_FAILURES = "explore-failures.txt"

WALK_CMD = "walk_packages.py {package} --output {output} --failures walk-failures.txt"
EXPLORE_CMD = "explore_packages.py {modules} --graph graph/{label}.json --dependencies deps/{label}"
TIMEOUT = 300

# Ignore packages that start with any of these strings
BLACKLIST_PREFIX = [
    "antigravity",
    "kite",
    "IPython",
    "appletrunner",
    "macropy.console",
    "plat-",
    "pyinotify",        # because it cause explore_packages to segfault
    "keyring",          # because it requires user input
    "keystoneclient",   # somehow causes the whole import loop to abort!
    "glanceclient",     # requires user input
    "novaclient",       # requires user input
    "twisted",          # because it hangs
    ".",
]

# Ignore packages that contain any of these strings
BLACKLIST_PATTERN = [
    "pygame.examples.",
    "pygame.tests.",
    "cherrypy.test."
]

# These packages are importable but will not be found by pkgutil
EXTRA_PACKAGES = [
    "__builtin__",
    "exceptions",
    "posix",
]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("packages", nargs="*", help="top-level packages to explore")
    parser.add_argument("--filter", nargs="?", help="regex filter for top-level package names")
    args = parser.parse_args()

    # Create a temp dir
    tempdir = tempfile.mkdtemp()
    print("created temp dir at "+tempdir)
    os.makedirs("logs/explore")
    os.makedirs("logs/walk")
    os.mkdir("graph")
    os.mkdir("deps")

    # Either use packages from command line or iterate through all top-level packages
    packages = args.packages
    if not packages:
        packages = EXTRA_PACKAGES
        for _, name, _ in pkgutil.iter_modules():
            if not any(name.startswith(prefix) for prefix in BLACKLIST_PREFIX):
                packages.append(name)

    # Filter by regex
    if args.filter:
        packages = filter(re.compile(args.filter).match, packages)

    # Sort packages
    packages = sorted(packages, key=str.lower)

    # Run explore_packages.py for every package
    num_successful = 0
    explore_failures = open(EXPLORE_FAILURES, "w")
    for package in packages:
        if package == '':
            continue

        print
        print("*" * 80)
        print("Starting:  " + package)

        # Get a list of sub-modules. This requires actually importing various packages
        # so do it in a subprocess.
        print("walking...")
        try:
            modules = list_submodules(package, tempdir)
        except Exception as ex:
            print("list_submodules failed")
            traceback.print_exc()
            continue

        # Filter out modules that contain any of the blacklisted patterns
        filtered = []
        for m in modules:
            if not any(pattern in m for pattern in BLACKLIST_PATTERN):
                filtered.append(m)
        modules = filtered

        if len(modules) == 0:
            print("did not find any submodules -- skipping.")
            continue

        print("exploring...")
        # Now explore the modules
        explore_command = EXPLORE_CMD.format(label=package, modules=" ".join(modules))
        code = run(explore_command, TIMEOUT, "logs/explore/{label}".format(label=package))

        if code == 0:
            num_successful += 1
        else:
            explore_failures.write(package + "\n")
            print("explore_packages failed")

        print("Complete: " + package)
        print("*" * 80)
        print

    explore_failures.close()

    try:
        shutil.rmtree(tempdir)
    except OSError:
        pass   # ignore errors here

    # If everything failed then return a non-zero exit code
    if num_successful == 0:
        sys.exit(1)


def list_submodules(package, tempdir):
    output_file = os.path.join(tempdir, package)

    # truncate the output file before each run
    try:
        os.remove(output_file)
    except OSError as e:
        pass

    walk_command = WALK_CMD.format(package=package, output=output_file)
    code = run(walk_command, TIMEOUT, "logs/walk/{label}".format(label=package))

    if code != 0:
        raise Exception("walk_packages failed")
    with open(output_file) as f:
        return f.read().split()


def kill_process(process):
    print("Timing out...")
    os.killpg(os.getpgid(process.pid), signal.SIGTERM)
    process.kill()


# Run command under a subprocess. Start a thread to track time.
# If command takes more than TIMEOUT_SEC seconds to run, kill it.
def run(cmd, timeout_sec, output):
    # use `script` to start a new tty to catch packages that write to `/dev/tty`
    cmd = """script -q -f {output} -c "{cmd}" """.format(cmd=cmd, output=output)
    process = subprocess.Popen(shlex.split(cmd), stdout=subprocess.PIPE,
        stderr=subprocess.PIPE, stdin=subprocess.PIPE, preexec_fn=os.setsid)
    timer = threading.Timer(timeout_sec, kill_process, [process])
    try:
        timer.start()
        stdout, stderr = process.communicate()
    finally:
        timer.cancel()
    return process.returncode

if __name__ == "__main__":
    main()
