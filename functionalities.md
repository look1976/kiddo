
I want to build a program which runs as a Windows 10/11 service so it's active since system bootup.
It's task is to control how much time my kids spend using their computer.
The idea is very simple.

A file in github repository contains entries in crontab alike format defining when my kids are allowed to log in and use their computers.
Example of entries (user, day of week, start hour - end hour)
Janek, Fri, 17-21
Igor, Sat, 13-17

Windows service checks this file for changes every minute and if change is detected it downloads it and applies locally.
Local application means:
Usernames, which are first parameters in the file, are only allowed to log in to Windows OS during defined times.
This is normally achieved by executing crafted "net user" command and scheduling a shutdown as a scheduled task.
PC is automatically shutdown once the "end hour" is reached.
Any other users except users from the file are not allowed and must be logged off and deleted.
So there must be a mechanism periodically checking for such condition.

