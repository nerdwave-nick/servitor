#!/usr/bin/env python3
"""A polkit agent for the whole logind session, as a desktop's graphical agent
registers itself, but asking upon its own terminal, where the driver answers.

pkttyagent will not serve: it registers for one process alone (its parent),
so run0 and pkexec called upon apart from any terminal would find no agent.
"""
import os

import gi

gi.require_version("Polkit", "1.0")
gi.require_version("PolkitAgent", "1.0")
from gi.repository import GLib, Polkit, PolkitAgent  # noqa: E402

subject = Polkit.UnixSession.new_for_process_sync(os.getpid(), None)
listener = PolkitAgent.TextListener.new(None)
handle = PolkitAgent.Listener.register(listener, PolkitAgent.RegisterFlags.NONE, subject,
                                       "/org/freedesktop/PolicyKit1/AuthenticationAgent", None)
print("the agent is registered for", subject.to_string(), flush=True)
GLib.MainLoop().run()
