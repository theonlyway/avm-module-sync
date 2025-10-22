package config

const (
	TempRepoPath       string = "./modules"
	ResourceModulesUrl string = "https://raw.githubusercontent.com/Azure/Azure-Verified-Modules/refs/heads/main/docs/static/module-indexes/TerraformResourceModules.csv"
	PatternModulesUrl  string = "https://raw.githubusercontent.com/Azure/Azure-Verified-Modules/refs/heads/main/docs/static/module-indexes/TerraformPatternModules.csv"
	UtilityModulesUrl  string = "https://raw.githubusercontent.com/Azure/Azure-Verified-Modules/refs/heads/main/docs/static/module-indexes/TerraformUtilityModules.csv"
)

var ProcessResourceModules bool
var ProcessPatternModules bool
var ProcessUtilityModules bool
var DebugMode bool
