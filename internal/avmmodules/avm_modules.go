package avmmodules

import (
	"fmt"
	"net/http"

	"github.com/gocarina/gocsv"
)

const (
	resourceModulesUrl string = "https://raw.githubusercontent.com/Azure/Azure-Verified-Modules/refs/heads/main/docs/static/module-indexes/TerraformResourceModules.csv"
	patternModulesUrl  string = "https://raw.githubusercontent.com/Azure/Azure-Verified-Modules/refs/heads/main/docs/static/module-indexes/TerraformPatternModules.csv"
	utilityModulesUrl  string = "https://raw.githubusercontent.com/Azure/Azure-Verified-Modules/refs/heads/main/docs/static/module-indexes/TerraformUtilityModules.csv"
)

type ResourceModulesStruct struct {
	ProviderNamespace       string `csv:"ProviderNamespace"`
	ResourceType            string `csv:"ResourceType"`
	ModuleDisplayName       string `csv:"ModuleDisplayName"`
	AlternativeNames        string `csv:"AlternativeNames"`
	ModuleName              string `csv:"ModuleName"`
	ParentModule            string `csv:"ParentModule"`
	ModuleStatus            string `csv:"ModuleStatus"`
	RepoURL                 string `csv:"RepoURL"`
	PublicRegistryReference string `csv:"PublicRegistryReference"`
	Description             string `csv:"Description"`
}

type PatternModulesStruct struct {
	ModuleDisplayName       string `csv:"ModuleDisplayName"`
	AlternativeNames        string `csv:"AlternativeNames"`
	ModuleName              string `csv:"ModuleName"`
	ModuleStatus            string `csv:"ModuleStatus"`
	RepoURL                 string `csv:"RepoURL"`
	PublicRegistryReference string `csv:"PublicRegistryReference"`
	Description             string `csv:"Description"`
}

type UtilityModulesStruct struct {
	ModuleDisplayName       string `csv:"ModuleDisplayName"`
	AlternativeNames        string `csv:"AlternativeNames"`
	ModuleName              string `csv:"ModuleName"`
	ModuleStatus            string `csv:"ModuleStatus"`
	RepoURL                 string `csv:"RepoURL"`
	PublicRegistryReference string `csv:"PublicRegistryReference"`
	Description             string `csv:"Description"`
}

func PrintResourceModules() {
	modules, err := getResourceModules()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	if len(modules) == 0 {
		fmt.Println("No resource modules found.")
		return
	}
	for _, module := range modules {
		fmt.Printf("ProviderNamespace: %s\nResourceType: %s\nModuleDisplayName: %s\nAlternativeNames: %s\nModuleName: %s\nParentModule: %s\nModuleStatus: %s\nRepoURL: %s\nPublicRegistryReference: %s\nDescription: %s\n\n",
			module.ProviderNamespace,
			module.ResourceType,
			module.ModuleDisplayName,
			module.AlternativeNames,
			module.ModuleName,
			module.ParentModule,
			module.ModuleStatus,
			module.RepoURL,
			module.PublicRegistryReference,
			module.Description,
		)
	}
}

func getResourceModules() ([]ResourceModulesStruct, error) {
	resp, err := http.Get(resourceModulesUrl)
	if err != nil {
		return nil, fmt.Errorf("error fetching resource modules: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	var modules []ResourceModulesStruct
	if err := gocsv.Unmarshal(resp.Body, &modules); err != nil {
		return nil, fmt.Errorf("error parsing CSV: %w", err)
	}
	return modules, nil
}

func getPatternModules() ([]PatternModulesStruct, error) {
	resp, err := http.Get(patternModulesUrl)
	if err != nil {
		return nil, fmt.Errorf("error fetching pattern modules: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	var modules []PatternModulesStruct
	if err := gocsv.Unmarshal(resp.Body, &modules); err != nil {
		return nil, fmt.Errorf("error parsing CSV: %w", err)
	}
	return modules, nil
}

func getUtilityModules() ([]UtilityModulesStruct, error) {
	resp, err := http.Get(utilityModulesUrl)
	if err != nil {
		return nil, fmt.Errorf("error fetching utility modules: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	var modules []UtilityModulesStruct
	if err := gocsv.Unmarshal(resp.Body, &modules); err != nil {
		return nil, fmt.Errorf("error parsing CSV: %w", err)
	}
	return modules, nil
}
