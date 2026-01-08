package avmmodules

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gocarina/gocsv"
	"github.com/theonlyway/avm-module-sync/internal/config"
	"go.uber.org/zap"
)

func getResourceModules(logger *zap.Logger) ([]ResourceModulesStruct, error) {
	var modules []ResourceModulesStruct
	if config.ReadLocalCsvFile {
		filePath := config.LocalCsvPath + "/TerraformResourceModules.csv"
		logger.Info("Reading resource modules from local CSV",
			zap.String("file_path", filePath))
		file, err := openCsvFile(filePath)
		if err != nil {
			logger.Error("Failed to open local resource modules CSV",
				zap.String("file_path", filePath),
				zap.Error(err))
			return nil, fmt.Errorf("error opening local resource modules CSV: %w", err)
		}
		defer file.Close()
		if err := gocsv.Unmarshal(file, &modules); err != nil {
			logger.Error("Failed to parse local resource modules CSV",
				zap.String("file_path", filePath),
				zap.Error(err))
			return nil, fmt.Errorf("error parsing local CSV: %w", err)
		}
		logger.Info("Successfully loaded resource modules from local CSV",
			zap.Int("module_count", len(modules)))
		return modules, nil
	}
	logger.Info("Fetching resource modules from remote URL",
		zap.String("url", config.ResourceModulesUrl))
	resp, err := http.Get(config.ResourceModulesUrl)
	if err != nil {
		logger.Error("Failed to fetch resource modules from remote URL",
			zap.String("url", config.ResourceModulesUrl),
			zap.Error(err))
		return nil, fmt.Errorf("error fetching resource modules: %w", err)
	}
	defer resp.Body.Close()
	logger.Info("Received HTTP response for resource modules",
		zap.Int("status_code", resp.StatusCode))
	if resp.StatusCode != http.StatusOK {
		logger.Error("Received non-200 HTTP response for resource modules",
			zap.Int("status_code", resp.StatusCode))
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}
	if err := gocsv.Unmarshal(resp.Body, &modules); err != nil {
		logger.Error("Failed to parse remote resource modules CSV",
			zap.Error(err))
		return nil, fmt.Errorf("error parsing CSV: %w", err)
	}
	logger.Info("Successfully loaded resource modules from remote URL",
		zap.Int("module_count", len(modules)))
	return modules, nil
}

// getPatternModules fetches and parses pattern modules from either a local CSV file or remote URL.
func getPatternModules(logger *zap.Logger) ([]PatternModulesStruct, error) {
	var modules []PatternModulesStruct
	if config.ReadLocalCsvFile {
		filePath := config.LocalCsvPath + "/TerraformPatternModules.csv"
		logger.Info("Reading pattern modules from local CSV",
			zap.String("file_path", filePath))
		file, err := openCsvFile(filePath)
		if err != nil {
			logger.Error("Failed to open local pattern modules CSV",
				zap.String("file_path", filePath),
				zap.Error(err))
			return nil, fmt.Errorf("error opening local pattern modules CSV: %w", err)
		}
		defer file.Close()
		if err := gocsv.Unmarshal(file, &modules); err != nil {
			logger.Error("Failed to parse local pattern modules CSV",
				zap.String("file_path", filePath),
				zap.Error(err))
			return nil, fmt.Errorf("error parsing local CSV: %w", err)
		}
		logger.Info("Successfully loaded pattern modules from local CSV",
			zap.Int("module_count", len(modules)))
		return modules, nil
	}
	logger.Info("Fetching pattern modules from remote URL",
		zap.String("url", config.PatternModulesUrl))
	resp, err := http.Get(config.PatternModulesUrl)
	if err != nil {
		logger.Error("Failed to fetch pattern modules from remote URL",
			zap.String("url", config.PatternModulesUrl),
			zap.Error(err))
		return nil, fmt.Errorf("error fetching pattern modules: %w", err)
	}
	defer resp.Body.Close()
	logger.Info("Received HTTP response for pattern modules",
		zap.Int("status_code", resp.StatusCode))
	if resp.StatusCode != http.StatusOK {
		logger.Error("Received non-200 HTTP response for pattern modules",
			zap.Int("status_code", resp.StatusCode))
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}
	if err := gocsv.Unmarshal(resp.Body, &modules); err != nil {
		logger.Error("Failed to parse remote pattern modules CSV",
			zap.Error(err))
		return nil, fmt.Errorf("error parsing CSV: %w", err)
	}
	logger.Info("Successfully loaded pattern modules from remote URL",
		zap.Int("module_count", len(modules)))
	return modules, nil
}

// getUtilityModules fetches and parses utility modules from either a local CSV file or remote URL.
func getUtilityModules(logger *zap.Logger) ([]UtilityModulesStruct, error) {
	var modules []UtilityModulesStruct
	if config.ReadLocalCsvFile {
		filePath := config.LocalCsvPath + "/TerraformUtilityModules.csv"
		logger.Info("Reading utility modules from local CSV",
			zap.String("file_path", filePath))
		file, err := openCsvFile(filePath)
		if err != nil {
			logger.Error("Failed to open local utility modules CSV",
				zap.String("file_path", filePath),
				zap.Error(err))
			return nil, fmt.Errorf("error opening local utility modules CSV: %w", err)
		}
		defer file.Close()
		if err := gocsv.Unmarshal(file, &modules); err != nil {
			logger.Error("Failed to parse local utility modules CSV",
				zap.String("file_path", filePath),
				zap.Error(err))
			return nil, fmt.Errorf("error parsing local CSV: %w", err)
		}
		logger.Info("Successfully loaded utility modules from local CSV",
			zap.Int("module_count", len(modules)))
		return modules, nil
	}
	logger.Info("Fetching utility modules from remote URL",
		zap.String("url", config.UtilityModulesUrl))
	resp, err := http.Get(config.UtilityModulesUrl)
	if err != nil {
		logger.Error("Failed to fetch utility modules from remote URL",
			zap.String("url", config.UtilityModulesUrl),
			zap.Error(err))
		return nil, fmt.Errorf("error fetching utility modules: %w", err)
	}
	defer resp.Body.Close()
	logger.Info("Received HTTP response for utility modules",
		zap.Int("status_code", resp.StatusCode))
	if resp.StatusCode != http.StatusOK {
		logger.Error("Received non-200 HTTP response for utility modules",
			zap.Int("status_code", resp.StatusCode))
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}
	if err := gocsv.Unmarshal(resp.Body, &modules); err != nil {
		logger.Error("Failed to parse remote utility modules CSV",
			zap.Error(err))
		return nil, fmt.Errorf("error parsing CSV: %w", err)
	}
	logger.Info("Successfully loaded utility modules from remote URL",
		zap.Int("module_count", len(modules)))
	return modules, nil

}

// openCsvFile opens and returns a file handle for the specified CSV file path.
func openCsvFile(path string) (*os.File, error) {
	return os.Open(path)
}
