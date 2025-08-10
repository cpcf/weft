package render

import (
	"fmt"
	"io/fs"
	"regexp"
	"slices"
	"strings"
	"text/template"
)

type BlockManager struct {
	templateFS fs.FS
	blocks     map[string]string
	overrides  map[string]string
	funcMap    template.FuncMap
}

type Block struct {
	Name        string `json:"name"`
	Content     string `json:"content"`
	DefaultOnly bool   `json:"default_only"`
	Override    bool   `json:"override"`
}

func NewBlockManager(templateFS fs.FS, funcMap template.FuncMap) *BlockManager {
	bm := &BlockManager{
		templateFS: templateFS,
		blocks:     make(map[string]string),
		overrides:  make(map[string]string),
		funcMap:    funcMap,
	}

	if bm.funcMap == nil {
		bm.funcMap = make(template.FuncMap)
	}

	bm.funcMap["block"] = bm.blockFunc
	bm.funcMap["override"] = bm.overrideFunc
	bm.funcMap["hasBlock"] = bm.hasBlockFunc
	bm.funcMap["blockContent"] = bm.blockContentFunc

	return bm
}

func (bm *BlockManager) blockFunc(name, defaultContent string) string {
	if override, exists := bm.overrides[name]; exists {
		return override
	}

	if content, exists := bm.blocks[name]; exists {
		return content
	}

	return defaultContent
}

func (bm *BlockManager) overrideFunc(name, content string) string {
	bm.overrides[name] = content
	return ""
}

func (bm *BlockManager) hasBlockFunc(name string) bool {
	_, exists := bm.blocks[name]
	if !exists {
		_, exists = bm.overrides[name]
	}
	return exists
}

func (bm *BlockManager) blockContentFunc(name string) string {
	if override, exists := bm.overrides[name]; exists {
		return override
	}

	if content, exists := bm.blocks[name]; exists {
		return content
	}

	return ""
}

func (bm *BlockManager) DefineBlock(name, content string) {
	bm.blocks[name] = content
}

func (bm *BlockManager) OverrideBlock(name, content string) {
	bm.overrides[name] = content
}

func (bm *BlockManager) GetBlock(name string) (string, bool) {
	if override, exists := bm.overrides[name]; exists {
		return override, true
	}

	content, exists := bm.blocks[name]
	return content, exists
}

func (bm *BlockManager) ListBlocks() []string {
	var names []string

	for name := range bm.blocks {
		names = append(names, name)
	}

	for name := range bm.overrides {
		found := slices.Contains(names, name)
		if !found {
			names = append(names, name)
		}
	}

	return names
}

func (bm *BlockManager) ProcessTemplate(templatePath string) (string, error) {
	content, err := fs.ReadFile(bm.templateFS, templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	processedContent := bm.preprocessBlocks(string(content))

	tmpl, err := template.New(templatePath).Funcs(bm.funcMap).Parse(processedContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", templatePath, err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, nil); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", templatePath, err)
	}

	return result.String(), nil
}

func (bm *BlockManager) preprocessBlocks(content string) string {
	defineRegex := regexp.MustCompile(`{{\s*define\s+"([^"]+)"\s*}}(.*?){{\s*end\s*}}`)
	blockRegex := regexp.MustCompile(`{{\s*block\s+"([^"]+)"\s+"([^"]*)"\s*}}(.*?){{\s*end\s*}}`)
	overrideRegex := regexp.MustCompile(`{{\s*override\s+"([^"]+)"\s*}}(.*?){{\s*end\s*}}`)

	content = defineRegex.ReplaceAllStringFunc(content, func(match string) string {
		groups := defineRegex.FindStringSubmatch(match)
		if len(groups) > 2 {
			bm.DefineBlock(groups[1], strings.TrimSpace(groups[2]))
		}
		return ""
	})

	content = blockRegex.ReplaceAllStringFunc(content, func(match string) string {
		groups := blockRegex.FindStringSubmatch(match)
		if len(groups) > 3 {
			blockName := groups[1]
			defaultContent := groups[2]
			blockContent := strings.TrimSpace(groups[3])

			if blockContent != "" {
				bm.DefineBlock(blockName, blockContent)
			}

			return fmt.Sprintf(`{{ block "%s" "%s" }}`, blockName, defaultContent)
		}
		return match
	})

	content = overrideRegex.ReplaceAllStringFunc(content, func(match string) string {
		groups := overrideRegex.FindStringSubmatch(match)
		if len(groups) > 2 {
			return fmt.Sprintf(`{{ override "%s" "%s" }}`, groups[1], strings.TrimSpace(groups[2]))
		}
		return match
	})

	return content
}

func (bm *BlockManager) LoadBlocksFromTemplate(templatePath string) error {
	content, err := fs.ReadFile(bm.templateFS, templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	blocks := bm.extractBlocks(string(content))
	for name, content := range blocks {
		bm.DefineBlock(name, content)
	}

	overrides := bm.extractOverrides(string(content))
	for name, content := range overrides {
		bm.OverrideBlock(name, content)
	}

	return nil
}

func (bm *BlockManager) extractBlocks(content string) map[string]string {
	blocks := make(map[string]string)

	defineRegex := regexp.MustCompile(`{{\s*define\s+"([^"]+)"\s*}}(.*?){{\s*end\s*}}`)
	matches := defineRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 2 {
			blocks[match[1]] = strings.TrimSpace(match[2])
		}
	}

	blockRegex := regexp.MustCompile(`{{\s*block\s+"([^"]+)"\s+"[^"]*"\s*}}(.*?){{\s*end\s*}}`)
	matches = blockRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 2 && strings.TrimSpace(match[2]) != "" {
			blocks[match[1]] = strings.TrimSpace(match[2])
		}
	}

	return blocks
}

func (bm *BlockManager) extractOverrides(content string) map[string]string {
	overrides := make(map[string]string)

	overrideRegex := regexp.MustCompile(`{{\s*override\s+"([^"]+)"\s*}}(.*?){{\s*end\s*}}`)
	matches := overrideRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 2 {
			overrides[match[1]] = strings.TrimSpace(match[2])
		}
	}

	return overrides
}

func (bm *BlockManager) GetFuncMap() template.FuncMap {
	return bm.funcMap
}

func (bm *BlockManager) ClearBlocks() {
	bm.blocks = make(map[string]string)
}

func (bm *BlockManager) ClearOverrides() {
	bm.overrides = make(map[string]string)
}

func (bm *BlockManager) ClearAll() {
	bm.ClearBlocks()
	bm.ClearOverrides()
}

func (bm *BlockManager) ValidateBlocks(templatePath string) error {
	content, err := fs.ReadFile(bm.templateFS, templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	blockRefs := bm.extractBlockReferences(string(content))

	for _, blockName := range blockRefs {
		if _, exists := bm.GetBlock(blockName); !exists {
			return fmt.Errorf("block '%s' is referenced but not defined", blockName)
		}
	}

	return nil
}

func (bm *BlockManager) extractBlockReferences(content string) []string {
	var refs []string

	blockRefRegex := regexp.MustCompile(`{{\s*block\s+"([^"]+)"`)
	matches := blockRefRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			refs = append(refs, match[1])
		}
	}

	return refs
}

func (bm *BlockManager) GetBlockInfo() []Block {
	var blocks []Block

	for name, content := range bm.blocks {
		block := Block{
			Name:        name,
			Content:     content,
			DefaultOnly: true,
			Override:    false,
		}

		if _, exists := bm.overrides[name]; exists {
			block.Override = true
			block.Content = bm.overrides[name]
		}

		blocks = append(blocks, block)
	}

	for name, content := range bm.overrides {
		if _, exists := bm.blocks[name]; !exists {
			blocks = append(blocks, Block{
				Name:        name,
				Content:     content,
				DefaultOnly: false,
				Override:    true,
			})
		}
	}

	return blocks
}

func (bm *BlockManager) LoadBlocksFromFile(blockFilePath string) error {
	content, err := fs.ReadFile(bm.templateFS, blockFilePath)
	if err != nil {
		return fmt.Errorf("failed to read block file %s: %w", blockFilePath, err)
	}

	return bm.LoadBlocksFromContent(string(content))
}

func (bm *BlockManager) LoadBlocksFromContent(content string) error {
	blocks := bm.extractBlocks(content)
	for name, blockContent := range blocks {
		bm.DefineBlock(name, blockContent)
	}

	overrides := bm.extractOverrides(content)
	for name, overrideContent := range overrides {
		bm.OverrideBlock(name, overrideContent)
	}

	return nil
}
