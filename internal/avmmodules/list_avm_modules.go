package avmmodules

import (
	"fmt"
	"net/http"
	"os"

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
	var modules []ResourceModulesStruct
	if config.ReadLocalCsvFile {
		filePath := config.LocalCsvPath + "/TerraformResourceModules.csv"
		file, err := openCsvFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("error opening local resource modules CSV: %w", err)
		}
		defer file.Close()
		if err := gocsv.Unmarshal(file, &modules); err != nil {
			return nil, fmt.Errorf("error parsing local CSV: %w", err)
		}
		return modules, nil
	}
	resp, err := http.Get(config.ResourceModulesUrl)
	if err != nil {
		return nil, fmt.Errorf("error fetching resource modules: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}
	if err := gocsv.Unmarshal(resp.Body, &modules); err != nil {
		return nil, fmt.Errorf("error parsing CSV: %w", err)
	}
	return modules, nil
}

func getPatternModules() ([]PatternModulesStruct, error) {
	var modules []PatternModulesStruct
	if config.ReadLocalCsvFile {
		filePath := config.LocalCsvPath + "/TerraformPatternModules.csv"
		file, err := openCsvFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("error opening local pattern modules CSV: %w", err)
		}
		defer file.Close()
		if err := gocsv.Unmarshal(file, &modules); err != nil {
			return nil, fmt.Errorf("error parsing local CSV: %w", err)
		}
		return modules, nil
	}
	resp, err := http.Get(config.PatternModulesUrl)
	if err != nil {
		return nil, fmt.Errorf("error fetching pattern modules: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}
	if err := gocsv.Unmarshal(resp.Body, &modules); err != nil {
		return nil, fmt.Errorf("error parsing CSV: %w", err)
	}
	return modules, nil
}

func getUtilityModules() ([]UtilityModulesStruct, error) {
	var modules []UtilityModulesStruct
	if config.ReadLocalCsvFile {
		filePath := config.LocalCsvPath + "/TerraformUtilityModules.csv"
		file, err := openCsvFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("error opening local utility modules CSV: %w", err)
		}
		defer file.Close()
		if err := gocsv.Unmarshal(file, &modules); err != nil {
			return nil, fmt.Errorf("error parsing local CSV: %w", err)
		}
		return modules, nil
	}
	resp, err := http.Get(config.UtilityModulesUrl)
	if err != nil {
		return nil, fmt.Errorf("error fetching utility modules: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}
	if err := gocsv.Unmarshal(resp.Body, &modules); err != nil {
		return nil, fmt.Errorf("error parsing CSV: %w", err)
	}
	return modules, nil

}

func openCsvFile(path string) (*os.File, error) {
	return os.Open(path)
}
