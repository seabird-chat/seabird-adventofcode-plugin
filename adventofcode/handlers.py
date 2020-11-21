import asyncio
import logging
import sys
from datetime import datetime
from urllib.parse import urlparse

import aiohttp
import betterproto

from .leaderboard import Leaderboard


TOP_SCORE_COUNT = 5
CHECK_STATUS_DELAY = 900


LOG = logging.getLogger("adventofcode")


class AdventOfCodeClient:
    def __init__(self, seabird, session_id, leaderboard_id, target_channel):
        aoc = aiohttp.ClientSession()
        aoc.cookie_jar.update_cookies({"session": session_id})

        self.aoc = aoc
        self.seabird = seabird
        self.leaderboard_id = leaderboard_id
        self.channel = target_channel

    def close(self):
        self.aoc.close()

    async def __aenter__(self):
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        self.close()

    async def stream_messages(self):
        async for message in self.seabird.stream_events():
            name, val = betterproto.which_one_of(message, "inner")
            if name == "command":
                asyncio.create_task(self.handle_command(val))

    async def handle_command(self, command):
        if command.command == "aoc" or command.command == "advent":
            arg = command.arg.strip()
            leaderboard = await self.lookup_leaderboard(event=arg)
            scores = [
                f"{member.name} ({member.local_score})"
                for member in leaderboard.members_by_score[:TOP_SCORE_COUNT]
            ]
            await self.seabird.send_message(
                channel_id=command.source.channel_id, text=", ".join(scores)
            )

    async def lookup_leaderboard(self, event=None):
        if not event:
            event = str(datetime.today().year)

        LOG.info("Looking up leaderboard for %s", event)

        async with self.aoc.get(
            f"https://adventofcode.com/{event}/leaderboard/private/view/{self.leaderboard_id}.json"
        ) as response:
            return Leaderboard(await response.json())

    async def check_status(self, f_name):
        try:
            with open(f_name, "r") as f:
                raw_timestamp = f.read()
                current_timestamp = int(raw_timestamp, base=10) if raw_timestamp else 0
        except FileNotFoundError:
            current_timestamp = 0

        while True:
            LOG.info("Checking leaderboard status")

            leaderboard = await self.lookup_leaderboard()
            events = [
                event for event in leaderboard.events
                if event.ts > current_timestamp
            ]

            if events:
                LOG.info("Found %d new event(s)", len(events))
            else:
                LOG.debug("Found 0 new event(s)")

            for event in events:
                await self.seabird.send_message(channel_id=self.channel, text=str(event))

                # It's arguably worse to cause a write on every message sent,
                # but this will make it possible to properly handle things if we
                # fail to send a message without having to start over.
                with open(f_name, "w") as f:
                    f.write(str(event.ts))

            LOG.info("Sleeping for %d seconds", CHECK_STATUS_DELAY)
            await asyncio.sleep(CHECK_STATUS_DELAY)
