#!/usr/bin/env python
import sys
from setuptools import find_packages, setup

requirements = []

if sys.version < '3':
    requirements.extend([
        'funcsigs',
    ])

setup(
    name='kite.pkgexploration',
    version='0.1.0',
    author='Manhattan Engineering Inc.',
    description='Kite Python Runtime Exploration',
    packages=find_packages(exclude=['tests']),
    install_requires=requirements,
    extras_require={'test': [
        'pytest',
        'django<2.0',
        'jsonschema',
        'numpy',
    ]}
)
