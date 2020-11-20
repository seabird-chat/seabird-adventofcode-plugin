import asyncio
import os

import seabird

from .handlers import AdventOfCodeClient


async def main():
    aoc_session = os.getenv("AOC_SESSION")
    if aoc_session is None:
        raise ValueError('Missing AOC_SESSION')

    aoc_leaderboard = os.getenv("AOC_LEADERBOARD")
    if aoc_leaderboard is None:
        raise ValueError('Missing AOC_LEADERBOARD')

    aoc_channel = os.getenv("AOC_CHANNEL")
    if aoc_channel is None:
        raise ValueError('Missing AOC_CHANNEL')

    ts_file_name = os.getenv("TIMESTAMP_FILE", "./aoc_timestamp.txt")

    async with seabird.Client(
        os.getenv("SEABIRD_HOST", "https://seabird-core.elwert.cloud"),
        os.getenv("SEABIRD_TOKEN"),
    ) as client:

        # Run a simple command to ensure we're connected. Otherwise, it will
        # wait until the stream events to tell us when we failed to connect and
        # the message will be wrong.
        await client.get_core_info()
        print("Connected to Seabird Core")

        async with AdventOfCodeClient(client, aoc_session, aoc_leaderboard, aoc_channel) as aoc:
            await asyncio.gather(
                aoc.stream_messages(),
                aoc.check_status(ts_file_name),
            )
