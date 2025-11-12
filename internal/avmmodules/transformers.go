package avmmodules

import "regexp"

// ModuleNameTransformer allows custom name transformation per module type
type ModuleNameTransformer func(string) string

func resourceNameTransformer(name string) string {
	var resourceRegex = regexp.MustCompile(`^(avm)-(res-)(.+)$`)
	if matches := resourceRegex.FindStringSubmatch(name); len(matches) == 4 {
		return "rvm-" + matches[2] + "azurerm-" + matches[3]
	}
	return name
}

func patternNameTransformer(name string) string {
	var patternRegex = regexp.MustCompile(`^avm-(ptn)-(.*)$`)
	if matches := patternRegex.FindStringSubmatch(name); len(matches) == 3 {
		return "rvm-pat-azurerm-" + matches[2]
	}
	return name
}

func utilityNameTransformer(name string) string {
	var utilityRegex = regexp.MustCompile(`^avm-(utl)-(.*)$`)
	if matches := utilityRegex.FindStringSubmatch(name); len(matches) == 3 {
		return "rvm-" + matches[1] + "-azurerm-" + matches[2]
	}
	return name
}
