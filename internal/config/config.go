package config

const (
	ResourceModulesUrl    string = "https://raw.githubusercontent.com/Azure/Azure-Verified-Modules/refs/heads/main/docs/static/module-indexes/TerraformResourceModules.csv"
	PatternModulesUrl     string = "https://raw.githubusercontent.com/Azure/Azure-Verified-Modules/refs/heads/main/docs/static/module-indexes/TerraformPatternModules.csv"
	UtilityModulesUrl     string = "https://raw.githubusercontent.com/Azure/Azure-Verified-Modules/refs/heads/main/docs/static/module-indexes/TerraformUtilityModules.csv"
	AdoEnterpriseAppId    string = "499b84ac-1321-427f-aa17-267ca6975798"
	AdoEnterpriseAppScope string = AdoEnterpriseAppId + "/.default"
	LocalCsvPath          string = "./third_party/module-indexes"
	BatchSize             int    = 10
	DefaultBranchName     string = "main"
)

var ProcessResourceModules bool
var ProcessPatternModules bool
var ProcessUtilityModules bool
var DebugMode bool
var CleanTempDirs bool
var UseLocalIdentity bool
var ReadLocalCsvFile bool
var PullRemoteTerraformRepository bool

var AdoOrganizationUrl string = "https://dev.azure.com/"
var AdoOrganization string
var AdoProject string
var AdoRepoId string
var AdoSessionToken string
var AdoPat string

var ModuleSyncAuthorName string
var ModuleSyncAuthorEmail string
var ModuleSyncSourceRepoChildPath string

var TempAvmModuleRepoPath string
var SourceRepoPath string

var AllowedStatuses []string
var OverrideModuleNames []string
