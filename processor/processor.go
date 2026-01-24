package processor

import (
	"encoding/json"
	"errors"
	"fmt"
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
	RootPath      string
}

// DefaultConfig returns the current default paths.
func DefaultConfig() Config {
	return Config{
		OutputPath:    "pages",
		TemplatesPath: filepath.Join(".processor", "templates"),
		SharedPath:    "shared",
		DataPath:      "data",
		RootPath:      "",
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
	if c.RootPath == "" {
		c.RootPath = defaults.RootPath
	}
	return c
}

func (c Config) resolvePaths() Config {
	if c.RootPath == "" {
		return c
	}

	c.OutputPath = resolvePath(c.RootPath, c.OutputPath)
	c.TemplatesPath = resolvePath(c.RootPath, c.TemplatesPath)
	c.SharedPath = resolvePath(c.RootPath, c.SharedPath)
	c.DataPath = resolvePath(c.RootPath, c.DataPath)
	return c
}

func resolvePath(root, path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

// Processor renders pages using Handlebars templates and structured data.
type Processor struct {
	Config    Config
	templates map[string]*raymond.Template
}

// New initializes a processor with the provided configuration.
func New(config Config) *Processor {
	return &Processor{Config: config.withDefaults().resolvePaths()}
}

func (p *Processor) Process() error {
	log.Info("Loading templates...")
	templates, err := loadTemplates(p.Config.TemplatesPath)
	if err != nil {
		return err
	}
	p.templates = templates

	shared, sharedErr := p.loadShared(p.Config.SharedPath)
	log.Info("Navigating data directory: %s\n", p.Config.DataPath)
	dataErr := p.convertPath(p.Config.DataPath, shared)
	return errors.Join(sharedErr, dataErr)
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

func (p *Processor) loadShared(path string) (map[string]interface{}, error) {
	log.Info("Loading shared from %s\n", path)
	shared := make(map[string]interface{})
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Info("Shared path is not found, ignoring loading\n")
		return shared, nil
	}

	var errs []error
	for _, file := range files {
		if file.IsDir() {
			nested, err := p.loadShared(filepath.Join(path, file.Name()))
			if err != nil {
				errs = append(errs, err)
			}
			shared[file.Name()] = nested
		} else {
			prefix := p.prefix(file.Name())
			content, err := p.loadFile(filepath.Join(path, file.Name()))
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if content != nil {
				shared[prefix] = content
			}
		}
	}
	return shared, errors.Join(errs...)
}

func (p *Processor) prefix(name string) string {
	return name[:len(name)-len(filepath.Ext(name))]
}

func (p *Processor) loadFile(file string) (map[string]interface{}, error) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		log.Info("Error reading file %s: %v\n", file, err)
		return nil, err
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
		return nil, nil
	}

	if err != nil {
		log.Info("Error parsing file %s: %v\n", file, err)
		return nil, err
	}
	return data, nil
}

func (p *Processor) convertPath(path string, shared map[string]interface{}) error {
	log.Info("Working with %s directory\n", path)
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Info("Path %s does not exist or it is not a directory\n", path)
		return err
	}

	var errs []error
	for _, file := range files {
		if file.IsDir() {
			if err := p.convertPath(filepath.Join(path, file.Name()), shared); err != nil {
				errs = append(errs, err)
			}
		} else {
			if err := p.renderFile(filepath.Join(path, file.Name()), shared); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

func (p *Processor) renderFile(file string, shared map[string]interface{}) error {
	log.Info("Rendering file %s\n", file)
	data, err := p.loadFile(file)
	if err != nil {
		return err
	}
	if data == nil {
		return nil
	}

	page, ok := data["_page"].(map[string]interface{})
	if !ok {
		log.Info("File %s does not contain _page directory\n", file)
		return nil
	}

	data["_shared"] = shared
	templateName, ok := page["template"].(string)
	if !ok {
		log.Info("File %s template is not defined\n", file)
		return nil
	}

	output, ok := page["output"].(string)
	if !ok {
		output = filepath.Base(file)
		log.Info("File %s output is not defined\n", file)
	}

	if output != "" && output[0] == '/' {
		output = output[1:]
	}

	outputPath := filepath.Join(p.Config.OutputPath, output)
	err = os.MkdirAll(filepath.Dir(outputPath), os.ModePerm)
	if err != nil {
		log.Info("Error creating directories for %s: %v\n", outputPath, err)
		return err
	}

	tmpl, ok := p.templates[templateName]
	if !ok {
		return fmt.Errorf("template %s is not found", templateName)
	}

	result, err := tmpl.Exec(data)
	if err != nil {
		return fmt.Errorf("execute template %s: %w", templateName, err)
	}

	log.Info("Writing output %s", outputPath)
	err = ioutil.WriteFile(outputPath, []byte(result), os.ModePerm)
	if err != nil {
		return fmt.Errorf("write output %s: %w", outputPath, err)
	}
	return nil
}
