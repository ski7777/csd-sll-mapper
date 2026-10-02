package config

import (
	"fmt"
	"os"
	"slices"

	"go.yaml.in/yaml/v4"
)

type Config struct {
	Events map[string]EventConfig `yaml:"events"`
}

type EventConfig struct {
	DwgFilePath   string        `yaml:"dwg_file_path"`
	ProductMapper ProductMapper `yaml:"product_mapper"`
}

func (e EventConfig) GetAllBlockNames() (names []string) {
	for _, pm := range e.ProductMapper {
		names = append(names, pm.PtxDwg.BlockNames...)
	}
	names = slices.Compact(names)
	return
}

type ProductMapper []ProductMapping

func (pm ProductMapper) GetPtxProductMapping(ptxProductID int, ptxVariantID *int) (mapping ProductMapping, ok bool) {
	for _, m := range pm {
		if m.PtxProduct.ID == ptxProductID && (m.PtxProduct.VariantID == nil || (ptxVariantID != nil && *m.PtxProduct.VariantID == *ptxVariantID)) {
			mapping = m
			ok = true
			return
		}
	}
	return
}

type ProductMapping struct {
	PtxProduct PtxProduct              `yaml:"ptx"`
	PtxDwg     PtxDwg                  `yaml:"ptxdwg"`
	PrideGuide PrideGuideProductExport `yaml:"prideguide"`
}

type PtxProduct struct {
	ID        int  `yaml:"id"`
	VariantID *int `yaml:"variant_id,omitempty"`
}

type PtxDwg struct {
	BlockNames       []string         `yaml:"block_names"`
	AttributeMapping AttributeMapping `yaml:"attribute_mapping"`
}

type AttributeMapping map[string]string // Pretix QuestionIdentifier -> DWG Attribute Name

type PrideGuideProductExport struct {
	NameQIdentifier string   `yaml:"name_qidentifier"`
	Category        string   `yaml:"category"`
	Subcategory     string   `yaml:"subcategory"`
	Shape           string   `yaml:"shape"`
	Width           float64  `yaml:"width"`
	Height          *float64 `yaml:"depth,omitempty"`
}

func NewConfigFromFile(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return &config, nil
}
