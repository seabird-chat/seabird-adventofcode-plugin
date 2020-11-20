import os

import seabird


async def main():
    async with seabird.Client(
        os.getenv("SEABIRD_HOST", "https://seabird-core.elwert.cloud"),
        os.getenv("SEABIRD_TOKEN"),
    ) as client:
        async for message in client.stream_events():
            print(message)
