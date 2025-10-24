package config

const (
	TempRepoPath          string = "./modules"
	ResourceModulesUrl    string = "https://raw.githubusercontent.com/Azure/Azure-Verified-Modules/refs/heads/main/docs/static/module-indexes/TerraformResourceModules.csv"
	PatternModulesUrl     string = "https://raw.githubusercontent.com/Azure/Azure-Verified-Modules/refs/heads/main/docs/static/module-indexes/TerraformPatternModules.csv"
	UtilityModulesUrl     string = "https://raw.githubusercontent.com/Azure/Azure-Verified-Modules/refs/heads/main/docs/static/module-indexes/TerraformUtilityModules.csv"
	AdoEnterpriseAppId    string = "499b84ac-1321-427f-aa17-267ca6975798"
	AdoEnterpriseAppScope string = AdoEnterpriseAppId + "/.default"
	LocalCsvPath          string = "./third_party/module-indexes"
)

var ProcessResourceModules bool
var ProcessPatternModules bool
var ProcessUtilityModules bool
var DebugMode bool
var CleanTempModulesDir bool
var UseLocalIdentity bool
var ReadLocalCsvFile bool

var AdoOrganizationUrl string = "https://dev.azure.com/"
var AdoOrganization string
var AdoProject string
var AdoRepo string
var AdoPat string
