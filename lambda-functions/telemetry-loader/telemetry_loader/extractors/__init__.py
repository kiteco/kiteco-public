import aioboto3
import io
import zlib

from telemetry_loader.streams.core import stream


def extract_s3(compressed=False, range_size=int(8e6)):
    @stream
    async def _extract_s3(bucket, key):
        offset = 0
        async with aioboto3.client('s3') as s3:
            if compressed:
                decompress = zlib.decompressobj(zlib.MAX_WBITS | 32)

            s3_meta = await s3.head_object(Bucket=bucket, Key=key)
            content_length = s3_meta['ContentLength']
            carryover = ''

            while offset <= content_length:
                range_end = offset + int(range_size) - 1
                s3_obj = await s3.get_object(Bucket=bucket, Key=key, Range='bytes={}-{}'.format(offset, range_end))
                offset = range_end + 1

                if compressed:
                    stream = decompress.decompress(await s3_obj['Body'].read())
                else:
                    stream = await s3_obj['Body'].read()

                lines = io.TextIOWrapper(io.BytesIO(stream))

                for line in lines:
                    if line.endswith('\n'):
                        if carryover:
                            line = carryover + line
                            carryover = ''
                        yield line.strip()
                    else:
                        carryover = line
                        break

    return _extract_s3
