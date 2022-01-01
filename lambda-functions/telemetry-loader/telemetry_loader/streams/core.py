import asyncio
import collections
import functools
import inspect


class Upstream(object):
    def __init__(self, run, cancel):
        self.run = run
        self.cancel = cancel

    def __iter__(self):
        return iter((self.run, self.cancel))

    def __or__(self, other):
        return other(self)

    def __ior__(self, other):
        called = other(self.run, self.cancel)
        self.run, self.cancel = called


async def cancel(upstream=None):
    if upstream:
        await upstream()


def make_stream_from_sync(iterable):
    async def data():
        for record in iterable:
            yield record
    return Upstream(data, cancel)


def make_stream_from_async(iterable):
    async def data():
        async for record in iterable:
            yield record
    return Upstream(data, cancel)


def stream(func_or_iterable):
    if hasattr(func_or_iterable, '__iter__'):
        return make_stream_from_sync(func_or_iterable)

    if hasattr(func_or_iterable, '__aiter__'):
        return make_stream_from_async(func_or_iterable)

    @functools.wraps(func_or_iterable)
    def wrapper(*args, **kwargs):
        return Upstream(lambda: func_or_iterable(*args, **kwargs), cancel)
    return wrapper


def _ensure_async_generator(func):
    if inspect.isasyncgenfunction(func):
        return func

    if inspect.iscoroutinefunction(func):
        raise TypeError("{} must not be a coroutine.".format(func))

    @functools.wraps(func)
    async def inner(it):
        async for record in it:
            res = func(record)
            if res != Skip:
                yield res
    return inner


def _pipe_helper(func):
    @functools.wraps(func)
    def wrapper(upstream_or_func):
        if inspect.isfunction(upstream_or_func):
            gen = _ensure_async_generator(upstream_or_func)
            return _pipe_helper(lambda x: gen(func(x)))
        return Upstream(lambda: func(upstream_or_func.run()), upstream_or_func.cancel)
    return wrapper


def pipe(func):
    return _pipe_helper(_ensure_async_generator(func))


Skip = pipe.Skip = object()


@pipe
def ident_pipe(rec):
    return rec


def accumulate(batch_size):
    @pipe
    async def _accumulate(iterator):
        buffer = []

        async for item in iterator:
            buffer.append(item)

            if len(buffer) >= batch_size:
                to_yield = buffer[:]
                buffer = []
                yield to_yield

        if buffer:
            yield buffer
    return _accumulate


def fork(count=2):
    def _fork(upstream):
        lazy_state = []

        def get_state():
            if not lazy_state:
                lazy_state.extend([upstream.run(), asyncio.Lock()])
            return lazy_state
        buffers = [collections.deque() for _ in range(count)]
        events = []
        done = object()

        def push(item):
            for deq in buffers:
                deq.append(item)
            for ev in events:
                ev.set()

        def one_fork(i, deq):
            async def run():
                it, lock = get_state()
                ev = asyncio.Event()
                events.append(ev)
                ev.set()
                while True:
                    await ev.wait()

                    while deq:
                        rec = deq.popleft()
                        if rec == done:
                            return
                        try:
                            yield rec
                        except Exception:
                            push(done)
                            raise

                    ev.clear()

                    async with lock:
                        if any(len(deq) > 0 for deq in buffers):
                            continue
                        try:
                            next_item = await it.__anext__()
                        except StopAsyncIteration:
                            next_item = done

                        push(next_item)

            return Upstream(run, upstream.cancel)

        return [one_fork(i, deq) for i, deq in enumerate(buffers)]
    return _fork


def decorator_with_options(func):
    @functools.wraps(func)
    def wrapper(*args, **kwargs):
        if len(args) == 1 and len(kwargs) == 0 and inspect.isfunction(args[0]):
            return func(args[0])

        def real_decorator(inner_func):
            return func(inner_func, *args, **kwargs)
        return real_decorator
    return wrapper


@decorator_with_options
def consumer(func, return_result=False):
    if not inspect.iscoroutinefunction(func):
        raise TypeError("Consumer function must be a coroutine.")

    @functools.wraps(func)
    def wrapper(upstream):
        async def run():
            res = await func(upstream.run())
            if return_result:
                return res
        return Upstream(run, upstream.cancel)
    return wrapper


@consumer
async def null_consumer(it):
    async for rec in it:
        pass


def consume(upstream):
    if callable(upstream):
        def the_thing(inner_upstream):
            return consume(upstream(inner_upstream))
        return the_thing

    async def data():
        return [rec async for rec in upstream.run()]

    return Upstream(data, upstream.cancel)


def map_stream(map_func):
    def consume(func_or_upstream):
        if callable(func_or_upstream):
            def inner(upstream):
                return consume(upstream(upstream))
            return inner

        async def run():
            async for rec in func_or_upstream.run():
                map_func(rec)

        return Upstream(run, func_or_upstream.cancel)
    return consume


def bundle(*streams):
    async def run():
        await asyncio.gather(*[strm.run() for strm in streams])
    return Upstream(run, streams[0].cancel)


@decorator_with_options
def side_effect(func):
    @functools.wraps(func)
    def wrapper(upstream):
        async def run_gen():
            async for line in upstream.run():
                se_res = func(line)
                if inspect.iscoroutine(se_res):
                    await se_res
                yield line

        async def run_coro():
            upstream_res = await upstream.run()
            se_res = func(upstream_res)
            if inspect.iscoroutine(se_res):
                await se_res
            return upstream_res

        return Upstream(run_gen if inspect.isasyncgenfunction(upstream.run) else run_coro, upstream.cancel)
    return wrapper
