package main

import (
	"encoding/json"
	"go/ast"
	"io/fs"
	"log"
	"os"
	"path"

	"github.com/BurntSushi/toml"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

type LocalizationFile struct {
	Path string
	Name string
	Lang language.Tag
	Ext  string
}

func GetLocalizationFiles() (files []LocalizationFile, err error) {
	dfs := os.DirFS(config.Directory)

	err = fs.WalkDir(dfs, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		matches := config.Pattern.FindStringSubmatch(name)
		if len(matches) == 0 {
			return NewError(ErrInvalidFilename, ErrorValueStr(name), ErrorWrapped(ErrFilenameDoesNotMatch))
		}

		if len(matches) != 4 {
			return NewError(ErrInvalidPattern, ErrorValueStr(config.Pattern.String()))
		}

		lang, err := language.Parse(matches[2])
		if err != nil {
			return NewError(ErrInvalidLanguage, ErrorValueStr(matches[1]), ErrorWrapped(err))
		}

		files = append(files, LocalizationFile{
			Path: path.Join(config.Directory, name),
			Name: matches[1],
			Lang: lang,
			Ext:  matches[3],
		})

		return nil
	})

	return files, err
}

func ReadLocalizationFiles(files []LocalizationFile) (locs []Localization, err error) {
	// Slice of sets of scope names
	// Each set corresponds to the localization at the same index
	var locsScopeNames []map[string]struct{}

	for _, file := range files {
		data, err := os.ReadFile(file.Path)
		if err != nil {
			return nil, NewError(ErrCouldNotReadFile, ErrorValueStr(file.Path), ErrorWrapped(err))
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
			return nil, NewError(ErrUnsupportedFileExtension, ErrorValueStr(file.Ext))
		}

		msgs, err := unmarshalMessages(data, unmarshaler)
		if err != nil {
			return nil, NewError(ErrCouldNotUnmarshalFile, ErrorValueStr(file.Path), ErrorWrapped(err))
		}

		scopes, err := processMessages(msgs)
		if err != nil {
			return nil, NewError(ErrCouldNotParseFile, ErrorValueStr(file.Path), ErrorWrapped(err))
		}

		locIdx := localizationIndex(locs, file.Lang)

		// If localization is not found, create it and add scopes to it
		if locIdx == -1 {
			locs = append(locs, Localization{
				Name:   file.Name,
				Lang:   file.Lang,
				Scopes: scopes,
			})

			locsScopeNames = append(locsScopeNames, make(map[string]struct{}))
			locScopeNames := locsScopeNames[len(locsScopeNames)-1]

			// Add scope names to the corresponding set so it is possible
			// to quickly check for duplicate messages
			for i := 0; i < len(scopes); i++ {
				locScopeNames[scopes[i].Name] = struct{}{}
			}

			continue
		}

		// If localization is found, check for duplicate messages
		// and add new messages

		loc := &locs[locIdx]
		locScopeName := &locsScopeNames[locIdx]

		for i := 0; i < len(scopes); i++ {
			scope := &scopes[i]

			if _, ok := (*locScopeName)[scope.Name]; ok {
				return nil, NewError(ErrInvalidLocalization,
					ErrorValueStr(loc.Lang.String()),
					NewDuplicateMessageError(scope.Name),
				)
			}

			(*locScopeName)[scope.Name] = struct{}{}
		}

		loc.Scopes = append(loc.Scopes, scopes...)
	}

	return locs, nil
}

// Checks whether different localizations contain all the same messages.
// Also checks if there are any localizations at all.
func CheckLocalizations(locs []Localization) (err error) {
	if len(locs) == 0 {
		return NewError(ErrNoLocalizationsFound)
	}

	// We consider the first localization as the "base" one
	baseLoc := &locs[0]
	// Set of localization messages of the base localization
	baseMsgs := make(map[string]struct{})

	for i := 0; i < len(baseLoc.Scopes); i++ {
		scope := &locs[0].Scopes[i]
		baseMsgs[scope.Name] = struct{}{}
	}

	for i := 1; i < len(locs); i++ {
		loc := &locs[i]
		// Set of localization messages
		msgs := make(map[string]struct{})

		// Add messages to the set and
		// check for unspecified messages in base localization
		for j := 0; j < len(loc.Scopes); j++ {
			scope := &loc.Scopes[j]

			if _, ok := baseMsgs[scope.Name]; !ok {
				return NewError(ErrInvalidLocalization,
					ErrorValueStr(baseLoc.Lang.String()),
					NewMessageNotSpecifiedError(scope.Name),
				)
			}

			msgs[scope.Name] = struct{}{}
		}

		// Check for unspecified messages in localization
		for msg := range baseMsgs {
			if _, ok := msgs[msg]; !ok {
				return NewError(ErrInvalidLocalization,
					ErrorValueStr(loc.Lang.String()),
					NewMessageNotSpecifiedError(msg),
				)
			}
		}
	}

	return nil
}

func generateFile(locFile *ast.File, filename string) (err error) {
	file, err := os.Create(filename)
	if err != nil {
		return NewError(ErrCouldNotCreateFile, ErrorValueStr(filename), ErrorWrapped(err))
	}
	defer file.Close()

	err = fprintAstFile(file, locFile)
	if err != nil {
		return NewError(ErrCouldNotWriteToFile, ErrorValueStr(filename), ErrorWrapped(err))
	}

	return nil
}

func GenerateLocalizations(locs []Localization) (err error) {
	locFiles := generateLocalizations(locs)

	err = os.MkdirAll(config.Output, 0755)
	if err != nil {
		return NewError(ErrCouldNotCreateDirectory, ErrorValueStr(config.Output), ErrorWrapped(err))
	}

	filenames := []string{path.Join(config.Output, "l10n.go")}

	for i := 1; i < len(locFiles); i++ {
		filename := locs[i-1].Name + "_" + locs[i-1].Lang.String() + ".go"
		filenames = append(filenames, path.Join(config.Output, filename))
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
	InitConfig()

	locFiles, err := GetLocalizationFiles()
	if err != nil {
		log.Fatal(err)
	}

	locs, err := ReadLocalizationFiles(locFiles)
	if err != nil {
		log.Fatal(err)
	}

	err = CheckLocalizations(locs)
	if err != nil {
		log.Fatal(err)
	}

	err = GenerateLocalizations(locs)
	if err != nil {
		log.Fatal(err)
	}
}
