package processor

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	log "github.com/andriyg76/glog"
	"github.com/aymerick/raymond"
	"gopkg.in/yaml.v2"
)

// Config describes where templates, data, and output live.
type Config struct {
	OutputPath    string
	TemplatesPath string
	SharedPath    string
	DataPath      string
}

// DefaultConfig returns the current default paths.
func DefaultConfig() Config {
	return Config{
		OutputPath:    "pages",
		TemplatesPath: filepath.Join(".processor", "templates"),
		SharedPath:    "shared",
		DataPath:      "data",
	}
}

func (c Config) withDefaults() Config {
	defaults := DefaultConfig()
	if c.OutputPath == "" {
		c.OutputPath = defaults.OutputPath
	}
	if c.TemplatesPath == "" {
		c.TemplatesPath = defaults.TemplatesPath
	}
	if c.SharedPath == "" {
		c.SharedPath = defaults.SharedPath
	}
	if c.DataPath == "" {
		c.DataPath = defaults.DataPath
	}
	return c
}

// Processor renders pages using Handlebars templates and structured data.
type Processor struct {
	Config    Config
	templates map[string]*raymond.Template
}

// New initializes a processor with the provided configuration.
func New(config Config) *Processor {
	return &Processor{Config: config.withDefaults()}
}

func (p *Processor) Process() {
	log.Info("Loading templates...")
	templates, err := loadTemplates(p.Config.TemplatesPath)
	if err != nil {
		log.Error("Can't load templates: %v", err)
		return
	}
	p.templates = templates

	shared := p.loadShared(p.Config.SharedPath)
	log.Info("Navigating data directory: %s\n", p.Config.DataPath)
	p.convertPath(p.Config.DataPath, shared)
}

func loadTemplates(templatesPath string) (map[string]*raymond.Template, error) {
	templates := map[string]*raymond.Template{}
	err := filepath.Walk(templatesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".hbs" {
			partialContent, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(templatesPath, path)
			if err != nil {
				return err
			}
			partialName := filepath.ToSlash(relPath[:len(relPath)-len(filepath.Ext(relPath))])

			raymond.RegisterPartial(partialName, string(partialContent))
			template, err := raymond.Parse(string(partialContent))
			if err != nil {
				return err
			}
			templates[partialName] = template
			log.Info("Template %s loaded...", partialName)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return templates, nil
}

func (p *Processor) loadShared(path string) map[string]interface{} {
	log.Info("Loading shared from %s\n", path)
	shared := make(map[string]interface{})
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Info("Shared path is not found, ignoring loading\n")
		return shared
	}

	for _, file := range files {
		if file.IsDir() {
			shared[file.Name()] = p.loadShared(filepath.Join(path, file.Name()))
		} else {
			prefix := p.prefix(file.Name())
			content := p.loadFile(filepath.Join(path, file.Name()))
			if content != nil {
				shared[prefix] = content
			}
		}
	}
	return shared
}

func (p *Processor) prefix(name string) string {
	return name[:len(name)-len(filepath.Ext(name))]
}

func (p *Processor) loadFile(file string) map[string]interface{} {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		log.Info("Error reading file %s: %v\n", file, err)
		return nil
	}

	var data map[string]interface{}
	switch filepath.Ext(file) {
	case ".json", ".json5":
		err = json.Unmarshal(content, &data)
	case ".yaml", ".yml":
		err = yaml.Unmarshal(content, &data)
	case ".toml":
		_, err = toml.Decode(string(content), &data)
	default:
		log.Info("Unknown file type %s\n", file)
		return nil
	}

	if err != nil {
		log.Info("Error parsing file %s: %v\n", file, err)
		return nil
	}
	return data
}

func (p *Processor) convertPath(path string, shared map[string]interface{}) {
	log.Info("Working with %s directory\n", path)
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Info("Path %s does not exist or it is not a directory\n", path)
		return
	}

	for _, file := range files {
		if file.IsDir() {
			p.convertPath(filepath.Join(path, file.Name()), shared)
		} else {
			p.renderFile(filepath.Join(path, file.Name()), shared)
		}
	}
}

func (p *Processor) renderFile(file string, shared map[string]interface{}) {
	log.Info("Rendering file %s\n", file)
	data := p.loadFile(file)
	if data == nil {
		return
	}

	page, ok := data["_page"].(map[string]interface{})
	if !ok {
		log.Info("File %s does not contain _page directory\n", file)
		return
	}

	data["_shared"] = shared
	templateName, ok := page["template"].(string)
	if !ok {
		log.Info("File %s template is not defined\n", file)
		return
	}

	output, ok := page["output"].(string)
	if !ok {
		output = filepath.Base(file)
		log.Info("File %s output is not defined\n", file)
	}

	if output[0] == '/' {
		output = output[1:]
	}

	outputPath := filepath.Join(p.Config.OutputPath, output)
	err := os.MkdirAll(filepath.Dir(outputPath), os.ModePerm)
	if err != nil {
		log.Info("Error creating directories for %s: %v\n", outputPath, err)
		return
	}

	tmpl, ok := p.templates[templateName]
	if !ok {
		log.Error("Template %s: is not found", templateName)
		return
	}

	result, err := tmpl.Exec(data)
	if err != nil {
		log.Error("Error executing template %s: %v\n", templateName, err)
		return
	}

	log.Info("Writing output %s", outputPath)
	err = ioutil.WriteFile(outputPath, []byte(result), os.ModePerm)
	if err != nil {
		log.Error("Error writing file %s: %v\n", outputPath, err)
	}
}
