import asyncio
import pytest

from telemetry_loader.streams.core import accumulate
from telemetry_loader.streams.core import consume
from telemetry_loader.streams.core import consumer
from telemetry_loader.streams.core import fork
from telemetry_loader.streams.core import pipe
from telemetry_loader.streams.core import stream
from telemetry_loader.streams.core import side_effect


pytestmark = pytest.mark.asyncio


async def test_pipe():
    @pipe
    def add_ten(record):
        return record + 10

    run, _ = consume(add_ten(stream([1, 2, 5])))
    assert await run() == [11, 12, 15]


async def test_pipe_iter():
    @pipe
    async def add_ten(it):
        async for i in it:
            yield i + 10

    run, _ = consume(add_ten(stream([1, 2, 5])))
    assert await run() == [11, 12, 15]


async def test_pipe_decorator():
    @pipe
    def add_ten(record):
        return record + 10

    @add_ten
    def add_ten_mul_two(record):
        return record * 2

    run, _ = consume(add_ten_mul_two(stream([1, 2, 5])))
    assert await run() == [22, 24, 30]

    @add_ten_mul_two
    async def add_ten_mul_two_add_1(it):
        async for record in it:
            yield record + 1

    run, _ = consume(add_ten_mul_two_add_1(stream([1, 2, 5])))
    assert await run() == [23, 25, 31]

    # The simple pipe function should not be a coroutine
    async def bad_fn(record):
        return record + 1

    with pytest.raises(TypeError):
        add_ten_mul_two(bad_fn)

    @add_ten_mul_two
    def add_ten_mul_two_add_4(record):
        return record + 4

    run, _ = consume(add_ten_mul_two_add_4(stream([1, 2, 5])))
    assert await run() == [26, 28, 34]


async def test_accumulate():
    run, _ = consume(accumulate(2)(stream([0, 1, 2])))
    assert await run() == [[0, 1], [2]]


async def test_fork():
    @pipe
    def add_ten(record):
        return record + 10

    strm1, strm2 = stream([0, 1, 2]) | fork()
    strm1 = strm1 | add_ten | consume
    strm2 = strm2 | consume

    tasks = [asyncio.create_task(strm()) for strm, _ in [strm1, strm2]]
    await asyncio.wait(tasks)
    assert [t.result() for t in tasks] == [[10, 11, 12], [0, 1, 2]]


async def test_fork_exception():
    class CustomException(Exception):
        pass

    @pipe
    def raise_exc(r):
        raise CustomException

    strm1, strm2 = stream([0, 1, 2]) | fork()
    strm1 = strm1 | raise_exc | consume
    strm2 = strm2 | consume

    tasks = [strm() for strm, _ in [strm1, strm2]]

    with pytest.raises(CustomException):
        await asyncio.gather(*tasks)


async def test_consumer():
    lines = []

    @consumer
    async def fn(it):
        result = [line async for line in it]
        lines.extend(result)
        return result

    run, _ = stream([0, 1, 2]) | fn
    assert await run() is None
    assert lines == [0, 1, 2]


async def test_consumer_return():
    lines = []

    @consumer(return_result=True)
    async def fn(it):
        result = [line async for line in it]
        lines.extend(result)
        return result

    run, _ = stream([0, 1, 2]) | fn
    assert await run() == [0, 1, 2]
    assert lines == [0, 1, 2]


async def test_side_effect():
    called = []

    @side_effect
    def effect(arg=None):
        called.append(arg)

    async_called = []

    @side_effect
    async def async_effect(arg=None):
        async_called.append(arg)

    run, _ = stream([0, 1, 2]) | effect | async_effect | consume
    assert await run() == [0, 1, 2]
    assert called == async_called == [0, 1, 2]

    run, _ = stream([0, 1, 2]) | consume | effect | async_effect
    assert await run() == [0, 1, 2]
    assert called[3:] == async_called[3:] == [[0, 1, 2]]
