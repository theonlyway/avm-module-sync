package main

import (
	"flag"
	"fmt"

	"github.com/theonlyway/avm-module-sync/internal/avmmodules"
)

func main() {
	var processResourceModules bool
	var ProcessPatternModules bool
	var ProcessUtilityModules bool

	flag.BoolVar(&processResourceModules, "process-resource", true, "Process Resource Modules")
	flag.BoolVar(&ProcessPatternModules, "process-pattern", false, "Process Pattern Modules")
	flag.BoolVar(&ProcessUtilityModules, "process-utility", false, "Process Utility Modules")
	flag.Parse()

	if processResourceModules {
		fmt.Println("Resource Modules:")
		err := avmmodules.ProcessResourceModules(func(module avmmodules.ResourceModulesStruct) {
			fmt.Printf("Module: %s, Type: %s, Status: %s, First Published In: %s\n", module.ModuleName, module.ResourceType, module.ModuleStatus, module.FirstPublishedIn)
		})
		if err != nil {
			fmt.Println("Error processing resource modules:", err)
		}
	}

	if ProcessPatternModules {
		fmt.Println("Pattern Modules:")
		err := avmmodules.ProcessPatternModules(func(module avmmodules.PatternModulesStruct) {
			fmt.Printf("Module: %s, Status: %s, First Published In: %s\n", module.ModuleName, module.ModuleStatus, module.FirstPublishedIn)
		})
		if err != nil {
			fmt.Println("Error processing pattern modules:", err)
		}
	}

	if ProcessUtilityModules {
		fmt.Println("Utility Modules:")
		err := avmmodules.ProcessUtilityModules(func(module avmmodules.UtilityModulesStruct) {
			fmt.Printf("Module: %s, Status: %s, First Published In: %s\n", module.ModuleName, module.ModuleStatus, module.FirstPublishedIn)
		})
		if err != nil {
			fmt.Println("Error processing utility modules:", err)
		}
	}
}
