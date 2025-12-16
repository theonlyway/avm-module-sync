package avmmodules

import (
	"context"

	"github.com/google/uuid"
	"github.com/theonlyway/avm-module-sync/internal/ado"
	"go.uber.org/zap"
)

type ModuleProcessor struct {
	Logger        *zap.Logger
	SugaredLogger *zap.SugaredLogger
	Clients       *ado.AdoClients
	Context       context.Context
	Project       string
	RepoId        *uuid.UUID
}

type ModulesStruct struct {
	ResourceModules []ResourceModulesStruct
	PatternModules  []PatternModulesStruct
	UtilityModules  []UtilityModulesStruct
}

type Module interface {
	GetRepoURL() string
	GetModuleName() string
}

func (m ResourceModulesStruct) GetRepoURL() string       { return m.RepoURL }
func (m PatternModulesStruct) GetRepoURL() string        { return m.RepoURL }
func (m UtilityModulesStruct) GetRepoURL() string        { return m.RepoURL }
func (m ResourceModulesStruct) GetModuleName() string    { return m.ModuleName }
func (m PatternModulesStruct) GetModuleName() string     { return m.ModuleName }
func (m UtilityModulesStruct) GetModuleName() string     { return m.ModuleName }
func (m ResourceModulesStruct) GetModuleStatus() string  { return m.ModuleStatus }
func (m PatternModulesStruct) GetModuleStatus() string   { return m.ModuleStatus }
func (m UtilityModulesStruct) GetModuleStatus() string   { return m.ModuleStatus }
