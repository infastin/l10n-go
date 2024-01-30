package main

import (
	"regexp"

	"github.com/alecthomas/kong"
)

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
	Dir     string `required:"" short:"d" type:"existingdir" placeholder:"DIR" help:"Path to the directory with localization files."`
	Pattern string `optional:"" short:"p" placeholder:"PATTERN" help:"Localization file regexp pattern."`
	Package string `required:"" short:"P" placeholder:"NAME" help:"Package name."`
	Output  string `required:"" short:"o" placeholder:"DIR" help:"Path to output directory."`
}

func InitConfig() {
	kong.Parse(&cli)

	if cli.Pattern == "" {
		cli.Pattern = `([a-z_]+)\.([a-z_]+)\.(yaml|yml|json|toml)`
	}

	config.Directory = cli.Dir
	config.Pattern = *regexp.MustCompile(cli.Pattern)
	config.PackageName = cli.Package
	config.Output = cli.Output

	config.FormatSpecifiers = []rune{'s', 'd', 'f', 'S', 'F', 'M'}
	config.SpecifierToGoType = DefaultSpecifiersToGoTypes()
}
