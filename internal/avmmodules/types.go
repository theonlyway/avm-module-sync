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

// GetRepoURL returns the repository URL for a resource module.
func (m ResourceModulesStruct) GetRepoURL() string       { return m.RepoURL }

// GetRepoURL returns the repository URL for a pattern module.
func (m PatternModulesStruct) GetRepoURL() string        { return m.RepoURL }

// GetRepoURL returns the repository URL for a utility module.
func (m UtilityModulesStruct) GetRepoURL() string        { return m.RepoURL }

// GetModuleName returns the module name for a resource module.
func (m ResourceModulesStruct) GetModuleName() string    { return m.ModuleName }

// GetModuleName returns the module name for a pattern module.
func (m PatternModulesStruct) GetModuleName() string     { return m.ModuleName }

// GetModuleName returns the module name for a utility module.
func (m UtilityModulesStruct) GetModuleName() string     { return m.ModuleName }

// GetModuleStatus returns the status of a resource module.
func (m ResourceModulesStruct) GetModuleStatus() string  { return m.ModuleStatus }

// GetModuleStatus returns the status of a pattern module.
func (m PatternModulesStruct) GetModuleStatus() string   { return m.ModuleStatus }

// GetModuleStatus returns the status of a utility module.
func (m UtilityModulesStruct) GetModuleStatus() string   { return m.ModuleStatus }
