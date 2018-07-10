# nsrecorder

`nsr` listens for events published to NSQ by [nspub](github.com/jw4/nspub), and records them.

The docker image expects a volume mounted at /var/lib/data in which it will create the sqlite db file, which defaults to nsr.db
