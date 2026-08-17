package catalog

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Armory extensions render manifest arguments as bare UPPERCASE words in Use
// and register no cobra.Args validator (see sliver client/command/extensions).
func TestBuildFromRoot_ExtensionCommandArguments(t *testing.T) {
	root := &cobra.Command{Use: "sliver"}

	ext := &cobra.Command{
		Use:   "nanodump PID DUMP-NAME WRITE-FILE SIGNATURE",
		Short: "A Beacon Object File that creates a minidump of the LSASS process.",
		Run:   func(cmd *cobra.Command, args []string) {},
	}
	f := pflag.NewFlagSet("nanodump", pflag.ContinueOnError)
	f.BoolP("save", "s", false, "Save output to disk")
	f.IntP("timeout", "t", 30, "command timeout in seconds")
	ext.Flags().AddFlagSet(f)

	root.AddCommand(ext)

	catalog, err := BuildFromRoot("session", root)
	if err != nil {
		t.Fatalf("BuildFromRoot: %v", err)
	}

	cmd := findCommand(t, catalog, "nanodump")

	want := []CommandArg{
		{Name: "PID", Required: true},
		{Name: "DUMP-NAME", Required: true},
		{Name: "WRITE-FILE", Required: true},
		{Name: "SIGNATURE", Required: true},
	}
	assertArgs(t, cmd.Arguments, want)

	if len(cmd.Flags) != 2 {
		t.Errorf("expected 2 flags (save, timeout), got %+v", cmd.Flags)
	}
}

// Mixed armory usage: bare required tokens alongside [OPTIONAL] brackets must
// all be collected, in order.
func TestBuildFromRoot_ExtensionMixedArguments(t *testing.T) {
	root := &cobra.Command{Use: "sliver"}
	ext := &cobra.Command{
		Use: "bof-servicemove TARGET [SERVICE]",
		Run: func(cmd *cobra.Command, args []string) {},
	}
	root.AddCommand(ext)

	catalog, err := BuildFromRoot("session", root)
	if err != nil {
		t.Fatalf("BuildFromRoot: %v", err)
	}

	cmd := findCommand(t, catalog, "bof-servicemove")
	want := []CommandArg{
		{Name: "TARGET", Required: true},
		{Name: "SERVICE", Required: false},
	}
	assertArgs(t, cmd.Arguments, want)
}

// Built-in sliver commands use <required> / [optional] placeholders — that
// behavior must not change.
func TestBuildFromRoot_BuiltinCommandArguments(t *testing.T) {
	root := &cobra.Command{Use: "sliver"}
	builtin := &cobra.Command{
		Use: "execute <command> [arguments...]",
		Run: func(cmd *cobra.Command, args []string) {},
	}
	root.AddCommand(builtin)

	catalog, err := BuildFromRoot("session", root)
	if err != nil {
		t.Fatalf("BuildFromRoot: %v", err)
	}

	cmd := findCommand(t, catalog, "execute")
	want := []CommandArg{
		{Name: "command", Required: true},
		{Name: "arguments", Required: false, Variadic: true},
	}
	assertArgs(t, cmd.Arguments, want)
}

// Commands whose usage contains no argument placeholders keep the existing
// inference behavior (probe cobra.Args).
func TestBuildFromRoot_InferredArgumentsUnchanged(t *testing.T) {
	root := &cobra.Command{Use: "sliver"}
	inferred := &cobra.Command{
		Use:  "cat",
		Run:  func(cmd *cobra.Command, args []string) {},
		Args: cobra.ExactArgs(1),
	}
	root.AddCommand(inferred)

	catalog, err := BuildFromRoot("session", root)
	if err != nil {
		t.Fatalf("BuildFromRoot: %v", err)
	}

	cmd := findCommand(t, catalog, "cat")
	want := []CommandArg{
		{Name: "argument", Required: true},
	}
	assertArgs(t, cmd.Arguments, want)
}

func findCommand(t *testing.T, catalog *CommandCatalog, name string) *CommandSchema {
	t.Helper()
	for i := range catalog.Groups {
		for j := range catalog.Groups[i].Commands {
			if catalog.Groups[i].Commands[j].Name == name {
				return &catalog.Groups[i].Commands[j]
			}
		}
	}
	t.Fatalf("command %q not found in catalog", name)
	return nil
}

func assertArgs(t *testing.T, got, want []CommandArg) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %d arguments, got %d: %+v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("argument %d: expected %+v, got %+v", i, want[i], got[i])
		}
	}
}
