#!/usr/bin/env python
import sys
from setuptools import find_packages, setup

setup(
    name='kite.metrics',
    version='0.1.0',
    author='Manhattan Engineering Inc.',
    description='Kite Metrics',
    packages=find_packages(),
    install_requires=[
        "jinja2>=2",
        "PyYAML>=5",
        "click>=7",
    ],
    entry_points = {
        'console_scripts': ['kite-metrics-schemas=kite_metrics.json_schema:main'],
    },
    python_requires='>=3.6',
    include_package_data = True,
)
