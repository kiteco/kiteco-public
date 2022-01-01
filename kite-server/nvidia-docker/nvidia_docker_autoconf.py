import os
import json
from py3nvml.py3nvml import *


def get_device_uuids():
    nvmlInit()
    deviceCount = nvmlDeviceGetCount()
    for i in range(deviceCount):
        h = nvmlDeviceGetHandleByIndex(i)
        yield nvmlDeviceGetUUID(h)


def configure(path='/etc/docker/daemon.json', key='NVIDIA-GPU'):
    try:
        with open(path) as f:
            conf = json.load(f)
    except:
        conf = {}

    resources = conf.get('node-generic-resources', [])
    resources = [r for r in resources if not r.startswith(key+'=')]
    for uuid in get_device_uuids():
        resources.append(key+'='+uuid)
    conf['node-generic-resources'] = resources

    with open(path, 'w') as f:
        json.dump(conf, f, indent=2)


def _parse_clocks_from_section(section):
    """ Output that is being parsed in this code:
==============NVSMI LOG==============

Timestamp                                 : Wed Oct 14 00:41:54 2020
Driver Version                            : 450.66
CUDA Version                              : 11.0

Attached GPUs                             : 1
GPU 00000000:00:1E.0
    Clocks
        Graphics                          : 1590 MHz
        SM                                : 1590 MHz
        Memory                            : 5000 MHz
        Video                             : 1470 MHz
    Applications Clocks
        Graphics                          : 1590 MHz
        Memory                            : 5001 MHz
    Default Applications Clocks
        Graphics                          : 585 MHz
        Memory                            : 5001 MHz
    Max Clocks
        Graphics                          : 1590 MHz
        SM                                : 1590 MHz
        Memory                            : 5001 MHz
        Video                             : 1470 MHz
    Max Customer Boost Clocks
        Graphics                          : 1590 MHz
    SM Clock Samples
        Duration                          : Not Found
        Number of Samples                 : Not Found
        Max                               : Not Found
        Min                               : Not Found
        Avg                               : Not Found
    Memory Clock Samples
        Duration                          : Not Found
        Number of Samples                 : Not Found
        Max                               : Not Found
        Min                               : Not Found
        Avg                               : Not Found
    Clock Policy
        Auto Boost                        : N/A
        Auto Boost Default                : N/A
    """
    s = os.popen('nvidia-smi -i 0 -q -d CLOCK')
    output = [x.strip() for x in s.readlines()]

    graphics, memory = "", ""
    in_section = False
    for line in output:
        if in_section:
            if 'Memory' in line:
                memory = line.split()[2]
            if 'Graphics' in line:
                graphics = line.split()[2]
            if len(memory) > 0 and len(graphics) > 0:
                return memory, graphics

        if line.startswith(section):
            in_section = True

    return None, None


def max_memory_graphics_clocks():
    return _parse_clocks_from_section('Max Clocks')


def current_memory_graphics_clocks():
    return _parse_clocks_from_section('Clocks')


def boost_clocks():
    c_memory, c_graphics = current_memory_graphics_clocks()
    if c_memory is None or c_graphics is None:
        print('[error] current memory & graphics clocks not found')
        return

    print(f'[info] current clocks, memory={c_memory}, graphics={c_graphics}')

    memory, graphics = max_memory_graphics_clocks()
    if memory is None or graphics is None:
        print('[error] max memory & graphics clocks not found')
        return

    print(f"[info] max clocks, memory={memory}, graphics={graphics}")

    print('[info] enabling persistence mode')
    ret = os.system('nvidia-smi -i 0 -pm 1')
    if ret != 0:
        return

    print('[info] setting memory & graphics clocks')
    ret = os.system(f"nvidia-smi -i 0 -ac {memory},{graphics}")
    if ret != 0:
        return

    c_memory, c_graphics = current_memory_graphics_clocks()
    if c_memory is None or c_graphics is None:
        print('[error] current memory & graphics clocks not found')
        return

    print(f'[info] updated clocks, memory={c_memory}, graphics={c_graphics}')


def main():
    configure()
    boost_clocks()


if __name__ == '__main__':
    main()
