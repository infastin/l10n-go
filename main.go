package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"os"
	"path"

	"github.com/BurntSushi/toml"
	"github.com/infastin/go-l10n/codegen"
	"github.com/infastin/go-l10n/common"
	"github.com/infastin/go-l10n/parse"
	"github.com/infastin/go-l10n/printer"
	"github.com/infastin/go-l10n/process"
	"github.com/infastin/go-l10n/scope"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

type LocalizationFile struct {
	Path     string
	Filename string
	Name     string
	Lang     language.Tag
	Ext      string
}

func GetLocalizationFiles() (files []LocalizationFile, err error) {
	entries, err := os.ReadDir(common.Config.Directory)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		matches := common.Config.Pattern.FindStringSubmatch(name)
		if len(matches) == 0 {
			return nil, common.NewError(common.ErrInvalidFilename,
				common.ErrorValueStr(name),
				common.ErrorWrapped(common.ErrFilenameDoesNotMatch),
			)
		}

		if len(matches) != 4 {
			return nil, common.NewError(common.ErrInvalidPattern,
				common.ErrorValueStr(common.Config.Pattern.String()),
			)
		}

		lang, err := language.Parse(matches[2])
		if err != nil {
			return nil, common.NewError(common.ErrInvalidLanguage,
				common.ErrorValueStr(matches[1]),
				common.ErrorWrapped(err),
			)
		}

		files = append(files, LocalizationFile{
			Path:     path.Join(common.Config.Directory, name),
			Filename: name,
			Name:     matches[1],
			Lang:     lang,
			Ext:      matches[3],
		})
	}

	return files, nil
}

func ReadLocalizationFiles(files []LocalizationFile) (locs []scope.Localization, err error) {
	// Slice of sets of scope names
	// Each set corresponds to the localization at the same index
	var locsScopeNames []map[string]struct{}

	for i := 0; i < len(files); i++ {
		file := &files[i]

		data, err := os.ReadFile(file.Path)
		if err != nil {
			return nil, common.NewError(common.ErrCouldNotReadFile,
				common.ErrorValueStr(file.Filename),
				common.ErrorWrapped(err),
			)
		}

		var unmarshaler func([]byte, any) error

		switch file.Ext {
		case "json":
			unmarshaler = json.Unmarshal
		case "yaml", "yml":
			unmarshaler = yaml.Unmarshal
		case "toml":
			unmarshaler = toml.Unmarshal
		default:
			return nil, common.NewError(common.ErrUnsupportedFileExtension, common.ErrorValueStr(file.Ext))
		}

		msgs, err := parse.UnmarshalMessages(data, unmarshaler)
		if err != nil {
			return nil, common.NewError(common.ErrCouldNotUnmarshalFile,
				common.ErrorValueStr(file.Filename),
				common.ErrorWrapped(err),
			)
		}

		mss, err := process.ProcessMessages(msgs)
		if err != nil {
			return nil, common.NewError(common.ErrCouldNotParseFile,
				common.ErrorValueStr(file.Filename),
				common.ErrorWrapped(err),
			)
		}

		locIdx := scope.LocalizationIndex(locs, file.Lang)

		// If localization is not found, create it and add scopes to it
		if locIdx == -1 {
			locs = append(locs, scope.Localization{
				Name:   file.Name,
				Lang:   file.Lang,
				Scopes: mss,
			})

			locsScopeNames = append(locsScopeNames, make(map[string]struct{}))
			locScopeNames := locsScopeNames[len(locsScopeNames)-1]

			// Add scope names to the corresponding set so it is possible
			// to quickly check for duplicate messages
			for i := 0; i < len(mss); i++ {
				locScopeNames[mss[i].Name] = struct{}{}
			}

			continue
		}

		// If localization is found, check for duplicate messages
		// and add new messages

		loc := &locs[locIdx]
		locScopeName := locsScopeNames[locIdx]

		for i := 0; i < len(mss); i++ {
			ms := &mss[i]

			if _, ok := locScopeName[ms.Name]; ok {
				return nil, common.NewError(common.ErrInvalidLocalization,
					common.ErrorValueStr(loc.Lang.String()),
					common.NewDuplicateMessageError(ms.Name),
				)
			}

			locScopeName[ms.Name] = struct{}{}
		}

		loc.Scopes = append(loc.Scopes, mss...)
	}

	return locs, nil
}

// Checks whether different localizations contain all the same messages.
// Also checks if there are any localizations at all.
func CheckLocalizations(locs []scope.Localization) (err error) {
	if len(locs) == 0 {
		return common.NewError(common.ErrNoLocalizationsFound)
	}

	// We consider the first localization as the "base" one
	baseLoc := &locs[0]
	// Set of localization messages of the base localization
	baseMsgs := make(map[string]struct{})

	for i := 0; i < len(baseLoc.Scopes); i++ {
		ms := &locs[0].Scopes[i]
		baseMsgs[ms.Name] = struct{}{}
	}

	for i := 1; i < len(locs); i++ {
		loc := &locs[i]
		// Set of localization messages
		msgs := make(map[string]struct{})

		// Add messages to the set and
		// check for unspecified messages in base localization
		for j := 0; j < len(loc.Scopes); j++ {
			ms := &loc.Scopes[j]

			if _, ok := baseMsgs[ms.Name]; !ok {
				return common.NewError(common.ErrInvalidLocalization,
					common.ErrorValueStr(baseLoc.Lang.String()),
					common.NewMessageNotSpecifiedError(ms.Name),
				)
			}

			msgs[ms.Name] = struct{}{}
		}

		// Check for unspecified messages in localization
		for msg := range baseMsgs {
			if _, ok := msgs[msg]; !ok {
				return common.NewError(common.ErrInvalidLocalization,
					common.ErrorValueStr(loc.Lang.String()),
					common.NewMessageNotSpecifiedError(msg),
				)
			}
		}
	}

	return nil
}

func generateFile(locFile *ast.File, filename string) (err error) {
	file, err := os.Create(filename)
	if err != nil {
		return common.NewError(common.ErrCouldNotCreateFile,
			common.ErrorValueStr(filename),
			common.ErrorWrapped(err),
		)
	}
	defer file.Close()

	err = printer.FprintAstFile(file, locFile)
	if err != nil {
		return common.NewError(common.ErrCouldNotWriteToFile,
			common.ErrorValueStr(filename),
			common.ErrorWrapped(err),
		)
	}

	return nil
}

func GenerateLocalizations(locs []scope.Localization) (err error) {
	locFiles := codegen.GenerateLocalizations(locs)

	err = os.MkdirAll(common.Config.Output, 0755)
	if err != nil {
		return common.NewError(common.ErrCouldNotCreateDirectory,
			common.ErrorValueStr(common.Config.Output),
			common.ErrorWrapped(err),
		)
	}

	filenames := []string{path.Join(common.Config.Output, "l10n.go")}

	for i := 1; i < len(locFiles); i++ {
		filename := locs[i-1].Name + "_" + locs[i-1].Lang.String() + ".go"
		filenames = append(filenames, path.Join(common.Config.Output, filename))
	}

	for i, locFile := range locFiles {
		err = generateFile(locFile, filenames[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	common.InitConfig()

	locFiles, err := GetLocalizationFiles()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	locs, err := ReadLocalizationFiles(locFiles)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	err = CheckLocalizations(locs)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	err = GenerateLocalizations(locs)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
}
