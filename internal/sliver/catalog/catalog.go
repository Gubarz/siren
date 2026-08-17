package catalog

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"siren/internal/sliver/console"
)

type CommandCatalog struct {
	Scope  string         `json:"scope"`
	Groups []CommandGroup `json:"groups"`
}

type CommandGroup struct {
	ID       string          `json:"id"`
	Title    string          `json:"title"`
	Commands []CommandSchema `json:"commands"`
}

type CommandSchema struct {
	Name        string        `json:"name"`
	Path        string        `json:"path"`
	Usage       string        `json:"usage"`
	Description string        `json:"description"`
	Arguments   []CommandArg  `json:"arguments"`
	Flags       []CommandFlag `json:"flags"`
	NeedsInput  bool          `json:"needsInput"`
	Supported   bool          `json:"supported"`
	Unavailable string        `json:"unavailable,omitempty"`
}

type CommandArg struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Variadic bool   `json:"variadic"`
}

type CommandFlag struct {
	Name      string `json:"name"`
	Shorthand string `json:"shorthand,omitempty"`
	Usage     string `json:"usage"`
	Type      string `json:"type"`
	Default   string `json:"default,omitempty"`
	Required  bool   `json:"required"`
	Boolean   bool   `json:"boolean"`
}

type Service struct {
	mu      sync.Mutex
	console *console.Service
}

func New(con *console.Service) *Service {
	return &Service{console: con}
}

func (s *Service) GetCommandCatalog(scope string) (*CommandCatalog, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	root, err := s.console.GetCommandRoot(scope)
	if err != nil {
		return nil, err
	}
	return BuildFromRoot(scope, root)
}

func BuildFromRoot(scope string, root *cobra.Command) (*CommandCatalog, error) {
	titles := make(map[string]string)
	for _, group := range root.Groups() {
		titles[group.ID] = group.Title
	}

	grouped := make(map[string][]CommandSchema)
	for _, command := range root.Commands() {
		if command.Hidden {
			continue
		}
		groupID := command.GroupID
		if groupID == "" {
			groupID = "other"
		}
		collectCommandSchemas(command, nil, groupID, grouped)
	}

	groupIDs := make([]string, 0, len(grouped))
	for id := range grouped {
		groupIDs = append(groupIDs, id)
	}
	sort.Slice(groupIDs, func(i, j int) bool {
		return groupTitle(groupIDs[i], titles) < groupTitle(groupIDs[j], titles)
	})

	catalog := &CommandCatalog{Scope: scope}
	for _, id := range groupIDs {
		commands := grouped[id]
		sort.Slice(commands, func(i, j int) bool {
			return commands[i].Path < commands[j].Path
		})
		catalog.Groups = append(catalog.Groups, CommandGroup{
			ID:       id,
			Title:    groupTitle(id, titles),
			Commands: commands,
		})
	}
	return catalog, nil
}

func collectCommandSchemas(
	command *cobra.Command,
	parentPath []string,
	groupID string,
	grouped map[string][]CommandSchema,
) {
	if command.Hidden {
		return
	}

	pathParts := append(append([]string{}, parentPath...), command.Name())
	if command.Run != nil || command.RunE != nil {
		path := strings.Join(pathParts, " ")
		usageTail := strings.TrimSpace(strings.TrimPrefix(command.Use, command.Name()))
		usage := path
		if usageTail != "" {
			usage += " " + usageTail
		}
		arguments := commandArguments(command, usageTail)
		flags := commandFlags(command)
		supported, unavailable := commandSupport(path)
		grouped[groupID] = append(grouped[groupID], CommandSchema{
			Name:        path,
			Path:        path,
			Usage:       usage,
			Description: commandDescription(command),
			Arguments:   arguments,
			Flags:       flags,
			NeedsInput:  len(arguments) > 0 || len(flags) > 0,
			Supported:   supported,
			Unavailable: unavailable,
		})
	}

	for _, child := range command.Commands() {
		collectCommandSchemas(child, pathParts, groupID, grouped)
	}
}

var (
	usageTokenPattern   = regexp.MustCompile(`<[^>]+>(\.\.\.)?|\[[^\]]+\](\.\.\.)?|\S+`)
	bareArgumentPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_-]*$`)
)

// commandArguments extracts positional arguments from the command's usage
// string. Sliver built-ins declare them as `<required>` / `[optional]`, while
// armory extensions and aliases render manifest arguments as bare UPPERCASE
// words (required) or `[UPPERCASE]` (optional).
func commandArguments(command *cobra.Command, usageTail string) []CommandArg {
	tokens := usageTokenPattern.FindAllString(usageTail, -1)
	arguments := make([]CommandArg, 0, len(tokens))
	for _, token := range tokens {
		variadic := strings.HasSuffix(token, "...")
		trimmed := strings.TrimSuffix(token, "...")
		var argument CommandArg
		switch {
		case strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, ">"):
			argument = CommandArg{Name: trimmed[1 : len(trimmed)-1], Required: true, Variadic: variadic}
		case strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]"):
			argument = CommandArg{Name: trimmed[1 : len(trimmed)-1], Variadic: variadic}
		case bareArgumentPattern.MatchString(trimmed):
			argument = CommandArg{Name: trimmed, Required: true, Variadic: variadic}
		default:
			continue
		}
		// Variadic markers may sit inside the brackets: `<args...>`, `[args...]`.
		if strings.HasSuffix(argument.Name, "...") {
			argument.Name = strings.TrimSuffix(argument.Name, "...")
			argument.Variadic = true
		}
		if argument.Name == "" {
			continue
		}
		arguments = append(arguments, argument)
	}
	if len(arguments) == 0 {
		return inferredCommandArguments(command)
	}
	return arguments
}

func inferredCommandArguments(command *cobra.Command) []CommandArg {
	if command.Args == nil {
		return nil
	}

	const probeLimit = 8
	accepted := probeAcceptedArgCounts(command, probeLimit)

	minimum, maximum := acceptedRange(accepted)
	if maximum <= 0 {
		return nil
	}

	unbounded := accepted[probeLimit]
	count := maximum
	if unbounded {
		count = minimum
		if count == 0 {
			count = 1
		}
	}
	return buildInferredArgs(count, minimum, unbounded)
}

func probeAcceptedArgCounts(command *cobra.Command, probeLimit int) []bool {
	accepted := make([]bool, probeLimit+1)
	for count := 0; count <= probeLimit; count++ {
		args := make([]string, count)
		for index := range args {
			args[index] = "value"
		}
		accepted[count] = command.Args(command, args) == nil
	}
	return accepted
}

func acceptedRange(accepted []bool) (int, int) {
	minimum := -1
	maximum := -1
	for count, valid := range accepted {
		if !valid {
			continue
		}
		if minimum == -1 {
			minimum = count
		}
		maximum = count
	}
	return minimum, maximum
}

func buildInferredArgs(count, minimum int, unbounded bool) []CommandArg {
	arguments := make([]CommandArg, 0, count)
	for index := 0; index < count; index++ {
		name := "argument"
		if count > 1 {
			name = fmt.Sprintf("argument-%d", index+1)
		}
		arguments = append(arguments, CommandArg{
			Name:     name,
			Required: index < minimum,
			Variadic: unbounded && index == count-1,
		})
	}
	return arguments
}

func commandSupport(path string) (bool, string) {
	if reason := console.NonInteractiveCommandReason(path); reason != "" {
		return false, reason
	}
	return true, ""
}

func commandFlags(command *cobra.Command) []CommandFlag {
	seen := make(map[string]bool)
	var flags []CommandFlag
	add := func(flag *pflag.Flag) {
		if flag.Hidden || seen[flag.Name] || flag.Name == "help" {
			return
		}
		seen[flag.Name] = true
		_, required := flag.Annotations[cobra.BashCompOneRequiredFlag]
		flags = append(flags, CommandFlag{
			Name:      flag.Name,
			Shorthand: flag.Shorthand,
			Usage:     flag.Usage,
			Type:      flag.Value.Type(),
			Default:   flag.DefValue,
			Required:  required,
			Boolean:   flag.NoOptDefVal == "true",
		})
	}

	command.NonInheritedFlags().VisitAll(add)
	command.InheritedFlags().VisitAll(add)
	sort.Slice(flags, func(i, j int) bool {
		return flags[i].Name < flags[j].Name
	})
	return flags
}

func commandDescription(command *cobra.Command) string {
	if description := strings.TrimSpace(command.Long); description != "" {
		return description
	}
	return strings.TrimSpace(command.Short)
}

func groupTitle(id string, titles map[string]string) string {
	if title := strings.TrimSpace(titles[id]); title != "" {
		return title
	}
	if id == "other" {
		return "Other"
	}
	return id
}
