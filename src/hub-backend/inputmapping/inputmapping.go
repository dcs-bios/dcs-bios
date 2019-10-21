// Package inputmapping manages a translation table
package inputmapping

import (
	"bufio"
	"fmt"
	"strings"

	"dcs-bios.a10c.de/dcs-bios-hub/configstore"
)

type CommandMatch struct {
	CommandMatch        string
	ArgumentMatch       string
	CommandReplacement  string
	ArgumentReplacement string
}

type InputMap map[string][]CommandMatch

type InputRemapper struct {
	unittypeMappings map[string]InputMap
	activeUnittype   string
}

func (ir *InputRemapper) SetActiveAircraft(name string) {
	ir.activeUnittype = name
}

func (ir *InputRemapper) LoadFromConfigStore() {
	file, err := configstore.OpenFile("inputmap.txt")
	if err != nil {
		fmt.Printf("could not load inputmap.txt: %v\n", err.Error())
		return
	}
	defer file.Close()

	ir.unittypeMappings = make(map[string]InputMap)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ",")
		cm := &CommandMatch{}
		unittype := parts[0]
		if len(parts) == 3 {
			cm.CommandMatch = parts[1]
			cm.CommandReplacement = parts[2]
		} else if len(parts) == 5 {
			cm.CommandMatch = parts[1]
			cm.ArgumentMatch = parts[2]
			cm.CommandReplacement = parts[3]
			cm.ArgumentReplacement = parts[4]
		} else {
			continue
		}

		if _, ok := ir.unittypeMappings[unittype]; !ok {
			ir.unittypeMappings[unittype] = make(InputMap)
		}
		ir.unittypeMappings[unittype][cm.CommandMatch] = append(ir.unittypeMappings[unittype][cm.CommandMatch], *cm)
	}
}

func (ir *InputRemapper) Remap(line string) string {
	mapping, ok := ir.unittypeMappings[ir.activeUnittype]
	if !ok {
		return line
	}
	parts := strings.Split(line, " ")
	if len(parts) != 2 {
		return line
	}
	msg, arg := parts[0], parts[1]

	for _, cmdMatch := range mapping[msg] {
		if cmdMatch.ArgumentMatch == arg || cmdMatch.ArgumentMatch == "" {
			argRep := arg
			if cmdMatch.ArgumentReplacement != "" {
				argRep = cmdMatch.ArgumentReplacement
			}
			// found replacement
			return cmdMatch.CommandReplacement + " " + argRep
		}
	}
	return line
}
