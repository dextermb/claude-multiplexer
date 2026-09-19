# Progress a work item as you work

When you work on a Jira or Linear item, keep its status in step with the work.
The multiplexer serves the work-item tools when a provider is configured. See
[../work-items.md](../work-items.md).

## Link the session

When you start work on an item, link the session to it.

- Call `mcp__cmux__set_workitem` with the item key, for example `GIM-1`.
- When more than one provider is configured, give the `provider` too.
- The multiplexer renames the session to the item key, so the row names the item.

## Move the status

Move the item status at each step of the work. Read the valid statuses first,
because the status names are the platform's own.

1. Call `mcp__cmux__list_workitem_statuses` to read the statuses the item may
   move to now.
2. Pick a status from that list. Do not invent a status, because the platform
   refuses a name it does not offer.
3. Call `mcp__cmux__set_workitem_status` with that status.

Move the status at these moments:

- When you start the work, move the item to the started status, for example
  `In Progress`.
- When you open a pull request, or the work waits for a review, move the item to
  the review status, for example `In Review`.
- When the work is merged and done, move the item to the done status, for
  example `Done`.

## Read and unlink

- Call `mcp__cmux__get_workitem` to read the current link and status.
- Call `mcp__cmux__unset_workitem` to drop the link when the item is no longer
  your work. The multiplexer clears the rename too.
