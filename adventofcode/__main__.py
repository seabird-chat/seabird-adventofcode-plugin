import asyncio

from dotenv import load_dotenv

from . import main


load_dotenv()
loop = asyncio.get_event_loop()
loop.run_until_complete(main())
