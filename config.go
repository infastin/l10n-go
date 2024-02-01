package main

import (
	"regexp"

	"github.com/alecthomas/kong"
)

const cliVersion = "v1.0.2"

var config struct {
	Directory         string
	PackageName       string
	Output            string
	Pattern           regexp.Regexp
	FormatSpecifiers  []rune
	SpecifierToGoType map[rune]GoType
	Imports           []GoImport
}

var cli struct {
	Dir     string           `required:"" short:"d" type:"existingdir" placeholder:"DIR" help:"Path to the directory with localization files."`
	Pattern string           `optional:"" short:"p" default:"${pattern}" placeholder:"PATTERN" help:"Localization file regexp pattern."`
	Package string           `optional:"" short:"P" default:"${package}" help:"Package name."`
	Output  string           `required:"" short:"o" placeholder:"DIR" help:"Path to output directory."`
	Version kong.VersionFlag `optional:"" short:"v" help:"Print version number."`
}

func InitConfig() {
	kong.Parse(&cli,
		kong.Description("Simple command-line utility to localize your Golang applications."),
		kong.Vars{
			"pattern": `([a-z_]+)\.([a-z_]+)\.(yaml|yml|json|toml)`,
			"package": "l10n",
			"version": cliVersion,
		},
	)

	config.Directory = cli.Dir
	config.Pattern = *regexp.MustCompile(cli.Pattern)
	config.PackageName = cli.Package
	config.Output = cli.Output

	config.FormatSpecifiers = []rune{'s', 'd', 'f', 'S', 'F', 'M'}
	config.SpecifierToGoType = DefaultSpecifiersToGoTypes()
}
