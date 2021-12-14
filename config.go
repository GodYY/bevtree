package bevtree

import (
	"encoding/xml"
	"os"
	"path"

	"github.com/pkg/errors"
)

type TreeEntry struct {
	Name string `xml:"name,attr"`
	Path string `xml:"path,attr"`
}

type Config struct {
	LoadAll     bool         `xml:"loadall"`
	TreeEntries []*TreeEntry `xml:"bevtrees>bevtree"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	decoder := xml.NewDecoder(file)

	xmlNameConfig := XMLName(XMLStringConfig)
	var cfgStart xml.StartElement
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}

		if start, ok := token.(xml.StartElement); ok && start.Name == xmlNameConfig {
			cfgStart = start
			break
		}
	}

	config := new(Config)
	if err := decoder.DecodeElement(config, &cfgStart); err != nil {
		return nil, errors.WithMessagef(err, "load config %s", path)
	}

	return config, nil
}

func SaveConfig(config *Config, path string) (err error) {
	if config == nil {
		return nil
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}

	defer func() {
		if e := file.Close(); e != nil {
			err = e
		}
	}()

	encoder := xml.NewEncoder(file)
	encoder.Indent("", indent)

	start := xml.StartElement{Name: XMLName(XMLStringConfig)}
	if err := encoder.EncodeElement(config, start); err != nil {
		return errors.WithMessagef(err, "save config %s", path)
	}

	return nil
}

func SaveConfigAndTrees(config *Config, trees map[string]*Tree, framework *Framework, rootPath, configPath string) error {
	for _, ta := range config.TreeEntries {
		tree := trees[ta.Name]
		if tree == nil {
			return errors.Errorf("tree \"%s\" not exist", ta.Name)
		}
	}

	if err := os.MkdirAll(rootPath, os.ModePerm); err != nil && !os.IsExist(err) {
		return err
	}

	for _, ta := range config.TreeEntries {
		tree := trees[ta.Name]

		treepath := path.Join(rootPath, ta.Path)
		dir := path.Dir(treepath)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil && !os.IsExist(err) {
			return err
		}

		if err := framework.EncodeXMLTreeFile(treepath, tree); err != nil {
			return err
		}
	}

	if err := SaveConfig(config, configPath); err != nil {
		return err
	}

	return nil
}
