import asyncio
import sys
from datetime import datetime
from urllib.parse import urlparse

import aiohttp
import betterproto

from .leaderboard import Leaderboard


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
        if command.command == "aoc":
            arg = command.arg.strip()
            leaderboard = await self.lookup_leaderboard(event=arg)
            scores = [
                f"{member.name} ({member.local_score})"
                for member in leaderboard.members_by_score[:5]
            ]
            await self.seabird.send_message(
                channel_id=command.source.channel_id, text=", ".join(scores)
            )

    async def lookup_leaderboard(self, event=None):
        if not event:
            event = str(datetime.today().year)

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
            print("Looking up current leaderboard")

            leaderboard = await self.lookup_leaderboard()
            events = list(filter(lambda event: event.ts > current_timestamp, leaderboard.events))

            if events:
                print(f"Found {len(events)} events")
            for event in events:
                await self.seabird.send_message(channel_id=self.channel, text=str(event))

                # It's arguably worse to cause a write on every message sent,
                # but this will make it possible to properly handle things if we
                # fail to send a message without having to start over.
                with open(f_name, "w") as f:
                    f.write(str(event.ts))

            await asyncio.sleep(900)
