import os
import mixpanel
import customerio

from telemetry_loader.streams.core import consumer


MIXPANEL_API_KEY = os.getenv('MIXPANEL_API_KEY')
MIXPANEL_TOKEN = os.getenv('MIXPANEL_TOKEN')

CUSTOMERIO_SITE_ID = os.getenv('CUSTOMERIO_SITE_ID')
CUSTOMERIO_API_KEY = os.getenv('CUSTOMERIO_API_KEY')

CIO_EVENTS = {'kite_status'}


@consumer
async def load_mixpanel(iterable):
    mp_client = mixpanel.Mixpanel(MIXPANEL_TOKEN)
    cio_client = customerio.CustomerIO(CUSTOMERIO_SITE_ID, CUSTOMERIO_API_KEY)

    async for line in iterable:
        user_id = line['user_id']
        name = line['name']

        # Customer.io requires user ID's to be non-empty ASCII
        user_id_err = not user_id or not all(ord(c) < 128 for c in user_id)

        try:
            if line['name'] in CIO_EVENTS and not user_id_err:
                cio_client.backfill(user_id, name, line['time'], **line)
            mp_client.track(user_id, name, line)
        except mixpanel.MixpanelException:
            ts = line.pop('time')
            mp_client.import_data(MIXPANEL_API_KEY, user_id, name, ts, line)
