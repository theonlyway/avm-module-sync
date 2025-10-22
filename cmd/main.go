package main

import (
	"flag"
	"fmt"

	"github.com/theonlyway/avm-module-sync/internal/avmmodules"
	"github.com/theonlyway/avm-module-sync/internal/config"
	"go.uber.org/zap"
)

func main() {

	flag.BoolVar(&config.ProcessResourceModules, "process-resource", true, "Process Resource Modules")
	flag.BoolVar(&config.ProcessPatternModules, "process-pattern", false, "Process Pattern Modules")
	flag.BoolVar(&config.ProcessUtilityModules, "process-utility", false, "Process Utility Modules")
	flag.BoolVar(&config.DebugMode, "debug", false, "Enable Debug Mode")

	logger, _ := zap.NewProduction()
	suagared := logger.Sugar()
	flag.Parse()

	processor := avmmodules.ModuleProcessor{Logger: logger, SugaredLogger: suagared}

	if config.ProcessResourceModules {
		logger.Info("Resource Modules:")
		err := processor.ProcessResourceModules(func(module avmmodules.ResourceModulesStruct) {
			logger.Info(
				"Processed resource module",
				zap.String("Module", module.ModuleName),
				zap.String("Status", module.ModuleStatus),
				zap.String("FirstPublishedIn", module.FirstPublishedIn),
			)
		})
		if err != nil {
			logger.Error("Error processing resource modules:", zap.Error(err))
		}
	}

	if config.ProcessPatternModules {
		fmt.Println("Pattern Modules:")
		err := avmmodules.ProcessPatternModules(func(module avmmodules.PatternModulesStruct) {
			fmt.Printf("Module: %s, Status: %s, First Published In: %s\n", module.ModuleName, module.ModuleStatus, module.FirstPublishedIn)
		})
		if err != nil {
			fmt.Println("Error processing pattern modules:", err)
		}
	}

	if config.ProcessUtilityModules {
		fmt.Println("Utility Modules:")
		err := avmmodules.ProcessUtilityModules(func(module avmmodules.UtilityModulesStruct) {
			fmt.Printf("Module: %s, Status: %s, First Published In: %s\n", module.ModuleName, module.ModuleStatus, module.FirstPublishedIn)
		})
		if err != nil {
			fmt.Println("Error processing utility modules:", err)
		}
	}
}
