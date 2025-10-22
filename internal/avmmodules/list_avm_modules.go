package avmmodules

import (
	"fmt"
	"net/http"

	"github.com/gocarina/gocsv"
	"github.com/theonlyway/avm-module-sync/internal/config"
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
	FirstPublishedIn        string `csv:"FirstPublishedIn"`
}

type PatternModulesStruct struct {
	ModuleDisplayName       string `csv:"ModuleDisplayName"`
	AlternativeNames        string `csv:"AlternativeNames"`
	ModuleName              string `csv:"ModuleName"`
	ModuleStatus            string `csv:"ModuleStatus"`
	RepoURL                 string `csv:"RepoURL"`
	PublicRegistryReference string `csv:"PublicRegistryReference"`
	Description             string `csv:"Description"`
	FirstPublishedIn        string `csv:"FirstPublishedIn"`
}

type UtilityModulesStruct struct {
	ModuleDisplayName       string `csv:"ModuleDisplayName"`
	AlternativeNames        string `csv:"AlternativeNames"`
	ModuleName              string `csv:"ModuleName"`
	ModuleStatus            string `csv:"ModuleStatus"`
	RepoURL                 string `csv:"RepoURL"`
	PublicRegistryReference string `csv:"PublicRegistryReference"`
	Description             string `csv:"Description"`
	FirstPublishedIn        string `csv:"FirstPublishedIn"`
}

func getResourceModules() ([]ResourceModulesStruct, error) {
	resp, err := http.Get(config.ResourceModulesUrl)
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
	resp, err := http.Get(config.PatternModulesUrl)
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
	resp, err := http.Get(config.UtilityModulesUrl)
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
