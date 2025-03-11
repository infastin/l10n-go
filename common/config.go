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

	anySpecifiers := [...]byte{'v'}
	intSpecifiers := [...]byte{'b', 'd', 'o', 'O', 'x', 'X'}
	floatSpecifiers := [...]byte{'f', 'F', 'e', 'E', 'g', 'G'}
	stringSpecifiers := [...]byte{'s', 'q'}
	stringerSpecifiers := [...]byte{'S'}

	for _, spec := range anySpecifiers {
		Config.FormatSpecifiers = append(Config.FormatSpecifiers, rune(spec))
		Config.SpecifierToGoType[spec] = ast.GoType{Type: "any"}
	}

	for _, spec := range intSpecifiers {
		Config.FormatSpecifiers = append(Config.FormatSpecifiers, rune(spec))
		Config.SpecifierToGoType[spec] = ast.GoType{Type: "int"}
	}

	for _, spec := range floatSpecifiers {
		Config.FormatSpecifiers = append(Config.FormatSpecifiers, rune(spec))
		Config.SpecifierToGoType[spec] = ast.GoType{Type: "float64"}
	}

	for _, spec := range stringSpecifiers {
		Config.FormatSpecifiers = append(Config.FormatSpecifiers, rune(spec))
		Config.SpecifierToGoType[spec] = ast.GoType{Type: "string"}
	}

	for _, spec := range stringerSpecifiers {
		Config.FormatSpecifiers = append(Config.FormatSpecifiers, rune(spec))
		Config.SpecifierToGoType[spec] = ast.GoType{
			Import:  "fmt",
			Package: "fmt",
			Type:    "Stringer",
		}
	}
}
