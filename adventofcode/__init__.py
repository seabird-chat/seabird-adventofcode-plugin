import asyncio
import logging
import os

import seabird

from .handlers import AdventOfCodeClient


LOG = logging.getLogger("adventofcode")


async def main():
    logging.basicConfig(
        format='[%(levelname)s %(asctime)s %(name)s] %(message)s',
        level=os.getenv("LOG_LEVEL", "INFO"),
    )

    aoc_session = os.environ["AOC_SESSION"]
    aoc_leaderboard = os.environ["AOC_LEADERBOARD"]
    aoc_channel = os.environ["AOC_CHANNEL"]

    ts_file_name = os.getenv("TIMESTAMP_FILE", "./aoc_timestamp.txt")

    async with seabird.Client(
        os.environ["SEABIRD_HOST"],
        os.environ["SEABIRD_TOKEN"],
    ) as client:

        # Run a simple command to ensure we're connected. Otherwise, it will
        # wait until the stream events to tell us when we failed to connect and
        # the message will be wrong.
        await client.get_core_info()
        LOG.info("Connected to Seabird Core")

        async with AdventOfCodeClient(client, aoc_session, aoc_leaderboard, aoc_channel) as aoc:
            await asyncio.gather(
                aoc.stream_messages(),
                aoc.check_status(ts_file_name),
                aoc.schedule_reminders(),
                aoc.schedule_gotime(),
            )
