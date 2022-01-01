import click
import os
import json

from kite_metrics.loader import load_json_schema
from jsonschema import validate

@click.command()
@click.argument('input', type=click.File('rb'))
def main(input):
    schema = json.loads(load_json_schema('kite_status'))
    for line in input:
        validate(instance=json.loads(line), schema=schema)


if __name__ == '__main__':
    main()
