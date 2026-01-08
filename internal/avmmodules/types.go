package avmmodules

import (
	"context"

	"github.com/google/uuid"
	"github.com/theonlyway/avm-module-sync/internal/ado"
	"go.uber.org/zap"
)

// ModuleProcessor holds the context and clients needed for processing AVM modules.
type ModuleProcessor struct {
	Logger        *zap.Logger
	SugaredLogger *zap.SugaredLogger
	Clients       *ado.AdoClients
	Context       context.Context
	Project       string
	RepoId        *uuid.UUID
	Modules       *ModulesStruct
}

// ModulesStruct holds all three types of AVM modules.
type ModulesStruct struct {
	ResourceModules []ResourceModulesStruct
	PatternModules  []PatternModulesStruct
	UtilityModules  []UtilityModulesStruct
}

// Module is a common interface implemented by all module types.
type Module interface {
	GetRepoURL() string
	GetModuleName() string
}

// ResourceModulesStruct represents an Azure Verified Module for Azure resources.
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

// PatternModulesStruct represents an Azure Verified Module for architectural patterns.
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

// UtilityModulesStruct represents an Azure Verified Module for utility functions.
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

// GetRepoURL returns the repository URL for a resource module.
func (m ResourceModulesStruct) GetRepoURL() string { return m.RepoURL }

// GetRepoURL returns the repository URL for a pattern module.
func (m PatternModulesStruct) GetRepoURL() string { return m.RepoURL }

// GetRepoURL returns the repository URL for a utility module.
func (m UtilityModulesStruct) GetRepoURL() string { return m.RepoURL }

// GetModuleName returns the module name for a resource module.
func (m ResourceModulesStruct) GetModuleName() string { return m.ModuleName }

// GetModuleName returns the module name for a pattern module.
func (m PatternModulesStruct) GetModuleName() string { return m.ModuleName }

// GetModuleName returns the module name for a utility module.
func (m UtilityModulesStruct) GetModuleName() string { return m.ModuleName }

// GetModuleStatus returns the status of a resource module.
func (m ResourceModulesStruct) GetModuleStatus() string { return m.ModuleStatus }

// GetModuleStatus returns the status of a pattern module.
func (m PatternModulesStruct) GetModuleStatus() string { return m.ModuleStatus }

// GetModuleStatus returns the status of a utility module.
func (m UtilityModulesStruct) GetModuleStatus() string { return m.ModuleStatus }
