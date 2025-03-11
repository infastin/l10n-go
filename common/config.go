package common

import (
	"regexp"

	"github.com/alecthomas/kong"
	"github.com/infastin/l10n-go/ast"
)

const cliVersion = "v1.0.6"

var Config struct {
	Directory         string
	PackageName       string
	Output            string
	Pattern           regexp.Regexp
	FormatSpecifiers  []rune
	SpecifierToGoType [255]ast.GoType
	Imports           []ast.GoImport
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

	Config.Directory = cli.Dir
	Config.Pattern = *regexp.MustCompile(cli.Pattern)
	Config.PackageName = cli.Package
	Config.Output = cli.Output

	Config.FormatSpecifiers = []rune{'v', 'd', 'f', 's', 'S'}

	Config.SpecifierToGoType['v'] = ast.GoType{Type: "any"}
	Config.SpecifierToGoType['d'] = ast.GoType{Type: "int"}
	Config.SpecifierToGoType['f'] = ast.GoType{Type: "float64"}
	Config.SpecifierToGoType['s'] = ast.GoType{Type: "string"}
	Config.SpecifierToGoType['S'] = ast.GoType{
		Import:  "fmt",
		Package: "fmt",
		Type:    "Stringer",
	}
}
