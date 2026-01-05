package avmmodules

import "regexp"

// ModuleNameTransformer is a function type that transforms module names according to specific patterns.
type ModuleNameTransformer func(string) string

// resourceNameTransformer transforms resource module names from AVM format to RVM format.
// Example: avm-res-compute-virtualmachine -> rvm-res-azurerm-compute-virtualmachine
func resourceNameTransformer(name string) string {
	var resourceRegex = regexp.MustCompile(`^(avm)-(res-)(.+)$`)
	if matches := resourceRegex.FindStringSubmatch(name); len(matches) == 4 {
		return "rvm-" + matches[2] + "azurerm-" + matches[3]
	}
	return name
}

// patternNameTransformer transforms pattern module names from AVM format to RVM format.
// Example: avm-ptn-network-hub -> rvm-pat-azurerm-network-hub
func patternNameTransformer(name string) string {
	var patternRegex = regexp.MustCompile(`^avm-(ptn)-(.*)$`)
	if matches := patternRegex.FindStringSubmatch(name); len(matches) == 3 {
		return "rvm-pat-azurerm-" + matches[2]
	}
	return name
}

// utilityNameTransformer transforms utility module names from AVM format to RVM format.
// Example: avm-utl-types -> rvm-utl-azurerm-types
func utilityNameTransformer(name string) string {
	var utilityRegex = regexp.MustCompile(`^avm-(utl)-(.*)$`)
	if matches := utilityRegex.FindStringSubmatch(name); len(matches) == 3 {
		return "rvm-" + matches[1] + "-azurerm-" + matches[2]
	}
	return name
}
