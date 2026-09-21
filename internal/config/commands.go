package config

// Command binds a key trigger to a script, so a press runs the script. Keys is
// the trigger, as one or two space-separated key names ("ctrl+g" or "b o").
// Label names the command in the help overlay and the notice. Script is the file
// to run, resolved the same as a custom bar element. See docs/config/commands.md.
type Command struct {
	Keys   string `json:"keys"`
	Label  string `json:"label"`
	Script string `json:"script"`
}
