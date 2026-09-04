# How a change reaches the screen

A tool that changes a live session moves the session itself, so the interface
learns of it the way it learns of everything else: the session emits an event,
and the bus carries it.

Two changes have no session event behind them, because both are only a write to
`meta.json`: archiving a stored session, and renaming one. For these the manager
publishes an event of its own, carrying `Notice` and `Reload`:

```
archive_session          Manager.Archive writes meta.json
                              |
                              v
                         Bus.Publish{Session:"landing", Reload:true,
                                     Notice:"docs archived landing"}
                              |
                              v
                         the interface reads the stored list again,
                         and shows the notice in the status bar
```

So the screen is true within one event, and the interface polls nothing.

`create_session` publishes a notice in the same way. The new session soon
streams its own events, but the notice names who created it and reloads the
list at once, so the row appears without a wait.

**A notice event carries no output and no streaming text.** The interface must
therefore keep it off the normal path, which treats empty streaming text as the
end of an answer and clears the pane. See `handleNotice` in
`internal/tui/app.go`.
