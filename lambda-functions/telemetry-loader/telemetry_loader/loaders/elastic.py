from operator import methodcaller
from telemetry_loader.streams.core import consumer

from elasticsearch.helpers import expand_action
from telemetry_loader.connections import connections_var


chunk_size = 500
max_chunk_bytes = 100 * 1024 * 1024


@consumer
async def load_elastic(it):
    client = connections_var.get().elasticsearch_async
    errors = []

    async for bulk_data, bulk_actions in _chunk_actions(it, chunk_size, max_chunk_bytes, client.transport.serializer):
        resp = await client.bulk("\n".join(bulk_actions) + "\n")
        # go through request-response pairs and detect failures
        for data, (op_type, item) in zip(
            bulk_data, map(methodcaller("popitem"), resp["items"])
        ):
            ok = 200 <= item.get("status", 500) < 300
            if not ok:
                # include original document source
                if len(data) > 1:
                    item["data"] = data[1]
                errors.append({op_type: item})

        if errors:
            raise Exception("%i document(s) failed to index." % len(errors), errors)


async def _chunk_actions(actions, chunk_size, max_chunk_bytes, serializer):
    """
    Split actions into chunks by number or size, serialize them into strings in
    the process.
    """
    bulk_actions, bulk_data = [], []
    size, action_count = 0, 0
    async for input_action in actions:
        action, data = expand_action(input_action)
        raw_data, raw_action = data, action
        action = serializer.dumps(action)
        # +1 to account for the trailing new line character
        cur_size = len(action.encode("utf-8")) + 1

        if data is not None:
            data = serializer.dumps(data)
            cur_size += len(data.encode("utf-8")) + 1

        # full chunk, send it and start a new one
        if bulk_actions and (
            size + cur_size > max_chunk_bytes or action_count == chunk_size
        ):
            yield bulk_data, bulk_actions
            bulk_actions, bulk_data = [], []
            size, action_count = 0, 0

        bulk_actions.append(action)
        if data is not None:
            bulk_actions.append(data)
            bulk_data.append((raw_action, raw_data))
        else:
            bulk_data.append((raw_action,))

        size += cur_size
        action_count += 1

    if bulk_actions:
        yield bulk_data, bulk_actions
