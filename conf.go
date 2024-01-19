package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
)

var currentConf *ReadformConf

const (
	ConfKeyReadwiseToken   = "readwise_token"
	ConfKeyReaderLocation  = "reader_location"
	ConfKeySaveFirstFetch  = "save_first_fetch"
	ConfKeyEnabledWebsites = "enabled_websites"
	ConfFile               = "data/conf.json"

	AgentConfUsername               = "username"
	AgentConfPassword               = "password"
	AgentConfKeyBlocklist           = "title_block_list"
	AgentConfKeyRSSLinks            = "rss_links"
	AgentConfLoginMethod            = "login_method"
	AgentConfIncludePremiumArticles = "include_premium_articles"

	GlobalConfigSectionName = "global"
	GlobalConfigDisplayName = "Global config"
)

type AgentConf struct {
	Username           string   `json:"username"`
	Password           string   `json:"password"`
	TitleBlockKeywords []string `json:"title_block_list"`
	RSSLinks           []string `json:"rss_links"`

	// Caixin
	IncludePremiumArticles bool `json:"include_premium_articles"`

	//FT
	LoginMethod string `json:"login_method"`
}

type ReadformConf struct {
	ReadwiseToken   string                `json:"readwise_token"`
	ReaderLocation  string                `json:"reader_location"`
	SaveFirstFetch  bool                  `json:"save_first_fetch"`
	EnabledWebsites []string              `json:"enabled_websites"`
	AgentConfs      map[string]*AgentConf `json:"agent"`
}

func NewReadformConf() *ReadformConf {
	return &ReadformConf{
		ReaderLocation: "feed",
		SaveFirstFetch: true,
		AgentConfs:     make(map[string]*AgentConf),
	}
}

// initConf initialize currentConf when Readform is started.
func initConf() {
	var err error
	currentConf, err = LoadConfFromFile()
	if err != nil {
		panic(fmt.Sprintf("LoadConfFromFile failed: %v", err))
	}
}

type ConfExport struct {
	ConfigSections []ConfigSection // defines all configurable items
}

type ConfigSection struct {
	Section     string     `json:"section"`      // should be agent name or GlobalConfigSectionName
	DisplayName string     `json:"display_name"` // displayed to user
	Configs     []ConfMeta `json:"configs"`      // config options under this section
}

type FieldType string

const (
	FieldTypeString     FieldType = "str"
	FieldTypeStringList FieldType = "str_list"
	FieldTypeCheckbox   FieldType = "multiple_selection"
	FieldTypeRadio      FieldType = "single_selection"
	FieldTypeBool       FieldType = "bool"

	FalseLiteral = "False"
	TrueLiteral  = "True"
)

type SelectOption struct {
	Value       string `json:"value"`
	DisplayName string `json:"display_name"`
	Selected    bool
}

type ConfMeta struct {
	ConfigName        string         `json:"config_name"`
	ConfigDescription string         `json:"config_description"`
	ConfigKey         string         `json:"config_key"`
	Type              FieldType      `json:"typ,omitempty"`
	DefaultValue      string         `json:"default_value,omitempty"`
	SelectOptions     []SelectOption `json:"selections,omitempty"`
	Required          bool           `json:"required,omitempty"`
	CurrentValue      string         `json:"current_value,omitempty"`
}

func (c *ReadformConf) Export() (ConfExport, error) {
	agentConfigs := make([]ConfigSection, 0, len(allAgents))
	agents := make([]SelectOption, 0, len(allAgents))

	for _, agent := range allAgents {
		// add current value to options
		agentConfValue := reflect.ValueOf(c.AgentConfs[agent.Name()])
		confOptions := slices.Clone(agent.ConfOptions())
		for i := range confOptions {
			// fill current value to CurrentValue field
			confKey := confOptions[i].ConfigKey
			confField := GetValueByJSONTag(agentConfValue, confKey)
			if confField.Kind() == reflect.String {
				confOptions[i].CurrentValue = confField.String()
			} else if confField.Kind() == reflect.Slice {
				// currently only []string is supported
				slice := make([]string, confField.Len())
				for i := 0; i < confField.Len(); i++ {
					slice[i] = confField.Index(i).String()
				}
				confOptions[i].CurrentValue = strings.Join(slice, ",")
			} else if confField.Kind() == reflect.Bool {
				if confField.Bool() {
					confOptions[i].CurrentValue = TrueLiteral
				} else {
					confOptions[i].CurrentValue = FalseLiteral
				}
			}
			if confOptions[i].CurrentValue == "" {
				confOptions[i].CurrentValue = confOptions[i].DefaultValue
			}

			// for radio and checkbox, also fill Selected
			for j := range confOptions[i].SelectOptions {
				if confOptions[i].SelectOptions[j].Value == confOptions[i].CurrentValue {
					confOptions[i].SelectOptions[j].Selected = true
					logger.Infof("111")
				}
			}
		}

		agentConfigs = append(agentConfigs, ConfigSection{
			Section:     agent.Name(),
			DisplayName: agent.DisplayName(),
			Configs:     confOptions,
		})
		agents = append(agents, SelectOption{
			Value:       agent.Name(),
			DisplayName: agent.DisplayName(),
			Selected:    slices.Contains(c.EnabledWebsites, agent.Name()),
		})
	}

	saveFirstBatch := FalseLiteral
	if c.SaveFirstFetch {
		saveFirstBatch = TrueLiteral
	}
	globalConfig := ConfigSection{
		Section:     GlobalConfigSectionName,
		DisplayName: GlobalConfigDisplayName,
		Configs: []ConfMeta{
			{
				ConfigName:        "Enabled websites",
				ConfigDescription: "The websites you want to enable. If a website is enabled here, please ensure to fill out all the required fields below on this website.",
				ConfigKey:         ConfKeyEnabledWebsites,
				Type:              FieldTypeCheckbox,
				SelectOptions:     agents,
				Required:          true,
			},
			{
				ConfigName:        "Readwise token",
				ConfigDescription: "Your Readwise token. Get one at https://readwise.io/access_token",
				ConfigKey:         ConfKeyReadwiseToken,
				Type:              FieldTypeString,
				Required:          true,
				CurrentValue:      c.ReadwiseToken,
			},
			{
				ConfigName:        "Readwise Reader location",
				ConfigDescription: "The location you would like to save to. Required. Default: feed.",
				ConfigKey:         ConfKeyReaderLocation,
				Type:              FieldTypeRadio,
				SelectOptions: []SelectOption{
					{Value: "feed", DisplayName: "Feed", Selected: c.ReaderLocation == "feed"},
					{Value: "new", DisplayName: "New", Selected: c.ReaderLocation == "new"},
					{Value: "later", DisplayName: "Later", Selected: c.ReaderLocation == "later"},
					{Value: "archive", DisplayName: "Archive", Selected: c.ReaderLocation == "archive"},
				},
				DefaultValue: "feed",
				Required:     true,
			},
			{
				ConfigName:        "Save first batch to Readwise Reader",
				ConfigDescription: "If to save first batch of articles to Readwise Reader after restarting Readform. Default: Yes.",
				ConfigKey:         ConfKeySaveFirstFetch,
				Type:              FieldTypeBool,
				Required:          true,
				CurrentValue:      saveFirstBatch,
			},
		},
	}

	// 将全局配置放置在配置列表的首位
	agentConfigs = append([]ConfigSection{globalConfig}, agentConfigs...)

	exportedConf := ConfExport{
		ConfigSections: agentConfigs,
	}
	return exportedConf, nil
}

// WriteDisk saves current conf to disk.
func (c *ReadformConf) WriteDisk() error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(ConfFile, data, 0644)
}

// LoadConfFromFile reads local conf file and return a pointer of ReadformConf. If conf file is not present,
// a default conf created by NewReadformConf will be returned.
func LoadConfFromFile() (*ReadformConf, error) {
	conf := NewReadformConf()
	data, err := os.ReadFile(ConfFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Config file not found, using default configuration.
			return conf, nil
		}
		return nil, err
	}
	err = json.Unmarshal(data, conf)
	if err != nil {
		// Config file is not valid JSON, it will be overwritten with default configuration.
		return conf, err
	}
	return conf, nil
}
