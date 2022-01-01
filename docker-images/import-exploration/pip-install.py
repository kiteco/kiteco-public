import subprocess
import sys

with open(sys.argv[1]) as packages:
    for package in packages:
        package = package.strip()
        print("trying..." + package)
        p = subprocess.Popen(["pip", "install", "-U", package], stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        stdout, stderr = p.communicate()
        if p.returncode != 0:
            print("*"*80)
            print("Could not install: " + package)
            print(stdout)
            print(stderr)

