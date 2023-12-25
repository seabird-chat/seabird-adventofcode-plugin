import asyncio
import logging
from datetime import datetime

import aiohttp
import betterproto
from aiofile import async_open
from dateutil.tz import gettz
from dateutil.relativedelta import relativedelta

from .leaderboard import Leaderboard


TOP_SCORE_COUNT = 5
CHECK_STATUS_DELAY = 900

AOC_TIMEZONE = gettz("US/Eastern")

LOG = logging.getLogger("adventofcode")


class AdventOfCodeClient:
    def __init__(self, seabird, session_id, leaderboard_id, target_channel):
        aoc = aiohttp.ClientSession(headers={"User-Agent": "seabird-adventofcode-plugin"})
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
            today = datetime.now(tz=AOC_TIMEZONE)
            if today.month == 12:
                event = str(today.year)
            else:
                event = str(today.year - 1)

        LOG.info("Looking up leaderboard for %s", event)

        async with self.aoc.get(
            f"https://adventofcode.com/{event}/leaderboard/private/view/{self.leaderboard_id}.json"
        ) as response:
            return Leaderboard(await response.json())

    async def check_status(self, f_name):
        try:
            async with async_open(f_name, "r") as f:
                raw_timestamp = await f.read()
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
                async with async_open(f_name, "w") as f:
                    await f.write(str(event.ts))

                current_timestamp = event.ts

            LOG.info("Sleeping for %d seconds", CHECK_STATUS_DELAY)
            await asyncio.sleep(CHECK_STATUS_DELAY)

    async def schedule_reminders(self):
        while True:
            # Determine the next notification time

            # On the last day of November and the first 24 days of December, we
            # want a reminder at 11:45 pm.

            next_reminder = None
            now = datetime.now(tz=AOC_TIMEZONE)
            day = 0

            if now.month == 12 and now.day >= 24 and now.hour >= 23 and now.minute >= 45:
                # We have to special case the end of December because the next
                # event is "next year".
                next_reminder = now + relativedelta(years=+1, month=11, day=31, hour=23, minute=45, second=0)
                day = 1
            elif now.month == 12 and now.hour >= 23 and now.minute >= 45:
                next_reminder = now + relativedelta(days=+1, hour=23, minute=45, second=0)
                day = now.day + 2
            elif now.month == 12:
                next_reminder = now + relativedelta(hour=23, minute=45, second=0)
                day = now.day + 1
            elif now.month == 11 and now.day >= 30 and now.hour >= 23 and now.minute >= 45:
                next_reminder = now + relativedelta(days=+1, hour=23, minute=45, second=0)
                day = 1
            else:
                next_reminder = now + relativedelta(month=11, day=31, hour=23, minute=45, second=0)
                day = 1

            sleep_secs = (next_reminder - now).total_seconds()
            LOG.info("Next reminder scheduled for %s - sleeping for %d seconds", next_reminder, sleep_secs)

            await asyncio.sleep(sleep_secs)

            msg = "Advent of Code day %d is starting in 15 minutes!" % day
            await self.seabird.send_message(channel_id=self.channel, text=str(msg))

    async def schedule_gotime(self):
        while True:
            # On the last day of November and the first 24 days of December, we
            # want a reminder at midnight.

            next_reminder = None
            now = datetime.now(tz=AOC_TIMEZONE)
            day = 0

            if now.month == 12 and now.day >= 25:
                # We have to special case the end of December because the next
                # event is "next year".
                next_reminder = now + relativedelta(years=+1, month=12, day=1, hour=0, minute=0, second=0)
                day = 1
            elif now.month == 12:
                next_reminder = now + relativedelta(days=+1, hour=0, minute=0, second=0)
                day = now.day + 1
            else:
                next_reminder = now + relativedelta(month=12, day=1, hour=0, minute=0, second=0)
                day = 1

            sleep_secs = (next_reminder - now).total_seconds()
            LOG.info("Next gotime scheduled for %s - sleeping for %d seconds", next_reminder, sleep_secs)

            await asyncio.sleep(sleep_secs)

            msg = "Advent of Code day %d is live!" % day
            await self.seabird.send_message(channel_id=self.channel, text=str(msg))
