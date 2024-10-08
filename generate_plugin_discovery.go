//go:build ignore

package main

import (
	"bufio"
	"fmt"
	"go/format"
	"log"
	"os"
	"strings"
)

const (
	pluginPath         = "github.com/xscaling/wing/plugins/"
	pluginFile         = "plugin.conf"
	pluginFSPath       = "plugins/" // Where the plugins are located on the file system
	header             = "// generated by generate_plugin_discovery.go; DO NOT EDIT\n\n"
	scalerFlagLine     = ">>> Scaler"
	replicatorFlagLine = ">>> Replicator"
)

type pluginInfo struct {
	name string
	repo string
}

type pluginMap map[string][]pluginInfo

func main() {
	pluginsMapping := pluginMap{
		"scaler":     make([]pluginInfo, 0),
		"replicator": make([]pluginInfo, 0),
	}

	file, err := os.Open(pluginFile)
	if err != nil {
		log.Fatalf("Failed to open %s: %q", pluginFile, err)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	flag := ""
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Check flag line first
		if line == scalerFlagLine {
			flag = "scaler"
			continue
		} else if line == replicatorFlagLine {
			flag = "replicator"
			continue
		}

		items := strings.Split(line, ":")
		if len(items) != 2 {
			// ignore empty lines
			continue
		}
		name, repo := items[0], items[1]

		for _, item := range pluginsMapping[flag] {
			if item.name == name {
				log.Fatalf("Duplicate entry %q", name)
			}
		}

		path := pluginPath + repo                               // Default, unless overridden by 3rd arg
		if _, err := os.Stat(pluginFSPath + repo); err != nil { // External package has been given
			path = repo
		}

		pluginsMapping[flag] = append(pluginsMapping[flag], pluginInfo{name, path})
	}
	genImports("core/engine/plugin/pluginz.go", "plugin", pluginsMapping)
	genDirectives("core/engine/pluginz.go", "engine", pluginsMapping)
}

func genImports(file, pack string, pluginsMapping pluginMap) {
	outs := header + "package " + pack + "\n\n" + "import ("

	if pluginCount := len(pluginsMapping["scaler"]) + len(pluginsMapping["replicator"]); pluginCount > 0 {
		outs += "\n"
	}

	outs += "\t// Include all plugins.\n"
	for _, plugins := range pluginsMapping {
		for _, plugin := range plugins {
			outs += `	_ "` + plugin.repo + `"` + "\n"
		}
	}
	outs += ")\n"

	if err := formatAndWrite(file, outs); err != nil {
		log.Fatalf("Failed to format and write: %q", err)
	}
}

func genDirectives(file, pack string, pluginsMapping pluginMap) {
	outs := `%spackage %s

	var (	
		Scalers = []string{%s}

		Replicators = []string{%s}
	)
`
	var (
		replicators []string
		scalers     []string
	)
	for _, plugin := range pluginsMapping["replicator"] {
		replicators = append(replicators, `"`+plugin.name+`"`)
	}
	for _, plugin := range pluginsMapping["scaler"] {
		scalers = append(scalers, `"`+plugin.name+`"`)
	}

	if err := formatAndWrite(file, fmt.Sprintf(outs,
		header, pack,
		strings.Join(scalers, ","),
		strings.Join(replicators, `,`),
	)); err != nil {
		log.Fatalf("Failed to format and write: %q", err)
	}
}

func formatAndWrite(file string, data string) error {
	res, err := format.Source([]byte(data))
	if err != nil {
		return err
	}

	if err = os.WriteFile(file, res, 0644); err != nil {
		return err
	}
	return nil
}
