from datetime import datetime
from functools import cached_property

from humanize import ordinal


class Leaderboard:
    def __init__(self, data):
        self.event = data["event"]
        self.members = [Member(self, member) for member in data["members"].values()]

    @cached_property
    def members_by_score(self):
        members = self.members
        members.sort(key=lambda x: x.local_score, reverse=True)
        return members

    @cached_property
    def events(self):
        ret = []
        for member in self.members:
            ret.extend(member.events)
        ret.sort(key=lambda x: x.ts)
        return ret

    def __str__(self):
        return self.event


class Member:
    def __init__(self, leaderboard, data):
        self.leaderboard = leaderboard
        self.name = data["name"]
        self.stars = data["stars"]
        self.local_score = data["local_score"]
        self.global_score = data["global_score"]

        self._completion_data = data["completion_day_level"]

    @cached_property
    def events(self):
        ret = []
        for day, stars in self._completion_data.items():
            for star, star_data in stars.items():
                ret.append(Event(self, day, star, star_data["get_star_ts"]))

        # Sort all the items then go through and update absolute_star
        ret.sort(key=lambda x: x.ts)
        for i, event in enumerate(ret, start=1):
            event.absolute_star = i

        return ret

    def __str__(self):
        return self.name


class Event:
    def __init__(self, member, day, star, ts):
        self.member = member
        self.day = day
        self.star = star
        self.ts = int(ts)
        self.absolute_star = 0

    def __str__(self):
        return (
            f"AoC {self.member.leaderboard.event}: "
            f"{self.member.name} completed "
            f"their {ordinal(self.absolute_star)} star, "
            f"the {ordinal(self.star)} star of day {self.day}, "
            f"at {datetime.fromtimestamp(self.ts).isoformat()} ({self.ts})"
        )
