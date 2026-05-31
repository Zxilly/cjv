package cli

import (
	"strings"
	"text/template"

	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/spf13/cobra"
)

const localizedHelpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`

const localizedUsageTemplate = `{{i18n "HelpUsageHeader"}}{{if .Runnable}}
  {{localizedUseLine .}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} {{commandPlaceholder}}{{end}}{{if gt (len .Aliases) 0}}

{{i18n "HelpAliasesHeader"}}
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

{{i18n "HelpExamplesHeader"}}
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

{{i18n "HelpAvailableCommandsHeader"}}{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

{{i18n "HelpAdditionalCommandsHeader"}}{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

{{i18n "HelpFlagsHeader"}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

{{i18n "HelpGlobalFlagsHeader"}}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

{{i18n "HelpAdditionalHelpTopicsHeader"}}{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

{{useCommandHelp .}}{{end}}
`

func configureCobraHelp(cmd *cobra.Command) {
	cobra.AddTemplateFuncs(template.FuncMap{
		"commandPlaceholder": commandPlaceholder,
		"i18n":               func(messageID string) string { return i18n.T(messageID, nil) },
		"localizedUseLine":   localizedUseLine,
		"useCommandHelp":     useCommandHelp,
	})
	cmd.SetHelpTemplate(localizedHelpTemplate)
	cmd.SetUsageTemplate(localizedUsageTemplate)
	localizeCobraHelpTree(cmd)
}

func localizeCobraHelpTree(cmd *cobra.Command) {
	cmd.InitDefaultHelpCmd()
	localizeGeneratedHelpCommand(cmd)

	cmd.InitDefaultHelpFlag()
	if flag := cmd.Flags().Lookup("help"); flag != nil {
		flag.Usage = i18n.T("HelpFlag", i18n.MsgData{"Command": helpTargetName(cmd)})
	}

	cmd.InitDefaultVersionFlag()
	if flag := cmd.Flags().Lookup("version"); flag != nil {
		flag.Usage = i18n.T("HelpVersionFlag", i18n.MsgData{"Command": helpTargetName(cmd)})
	}

	for _, sub := range cmd.Commands() {
		localizeCobraHelpTree(sub)
	}
}

func localizeGeneratedHelpCommand(cmd *cobra.Command) {
	for _, sub := range cmd.Commands() {
		if sub.Name() != "help" || sub.Parent() != cmd {
			continue
		}
		sub.Short = i18n.T("HelpCommandShort", nil)
		sub.Long = i18n.T("HelpCommandLong", i18n.MsgData{"Command": cmd.DisplayName()})
		return
	}
}

func localizedUseLine(cmd *cobra.Command) string {
	use := strings.Replace(cmd.Use, cmd.Name(), cmd.DisplayName(), 1)
	var useLine string
	if cmd.HasParent() {
		useLine = cmd.Parent().CommandPath() + " " + use
	} else {
		useLine = use
	}

	flagPlaceholder := i18n.T("HelpFlagsPlaceholder", nil)
	useLine = strings.ReplaceAll(useLine, "[flags]", flagPlaceholder)
	if !cmd.DisableFlagsInUseLine && cmd.HasAvailableFlags() && !strings.Contains(useLine, flagPlaceholder) {
		useLine += " " + flagPlaceholder
	}
	return useLine
}

func commandPlaceholder() string {
	return i18n.T("HelpCommandPlaceholder", nil)
}

func useCommandHelp(cmd *cobra.Command) string {
	return i18n.T("HelpUseCommandHelp", i18n.MsgData{
		"Command":            cmd.CommandPath(),
		"CommandPlaceholder": commandPlaceholder(),
	})
}

func helpTargetName(cmd *cobra.Command) string {
	if name := cmd.DisplayName(); name != "" {
		return name
	}
	return i18n.T("HelpThisCommand", nil)
}
