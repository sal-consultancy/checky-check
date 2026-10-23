package main

import "testing"

func TestActionReferences(t *testing.T) {
	c := Config{Actions: map[string]CorrectiveAction{"reboot": {Type: "command", Command: "true", Label: "Reboot", AllowedRoles: []string{"admin"}}}, Checks: map[string]Check{"test": {Actions: []ActionReference{{Action: "reboot", When: "failed"}}}}}
	if e := validateActions(c); e != nil {
		t.Fatal(e)
	}
	c.Checks["test"] = Check{Actions: []ActionReference{{Action: "missing", When: "failed"}}}
	if validateActions(c) == nil {
		t.Fatal("unknown action accepted")
	}
}

func TestAlwaysActionReference(t *testing.T) {
	c := Config{Actions: map[string]CorrectiveAction{"reboot": {Type: "command", Command: "true", Label: "Reboot", AllowedRoles: []string{"admin"}}}, Checks: map[string]Check{"test": {Actions: []ActionReference{{Action: "reboot", When: "always"}}}}}
	if err := validateActions(c); err != nil {
		t.Fatal(err)
	}
	c.Checks["test"] = Check{Actions: []ActionReference{{Action: "reboot", When: "passed"}}}
	if validateActions(c) == nil {
		t.Fatal("unsupported condition accepted")
	}
}
