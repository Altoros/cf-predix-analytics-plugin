package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/cli/cf/configuration/confighelpers"
)

func (p *AnalyticsPlugin) loadConfig() {
	home, err := confighelpers.DefaultFilePath()
	if err != nil {
		fmt.Printf("Loading config failed: %s\n", err)
		panic(1)
	}
	file := filepath.Join(filepath.Dir(home), "cf_predix_analytics_plugin")
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return
	}
	config, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Printf("Loading config failed: %s\n", err)
		panic(1)
	}
	if json.Unmarshal(config, p) != nil {
		fmt.Printf("Loading config failed: %s\n", err)
		panic(1)
	}
}

func (p *AnalyticsPlugin) saveConfig() {
	home, err := confighelpers.DefaultFilePath()
	if err != nil {
		fmt.Printf("Saving config failed: %s\n", err)
		panic(1)
	}
	file := filepath.Join(filepath.Dir(home), "cf_predix_analytics_plugin")
	config, err := json.Marshal(p)
	if err != nil {
		fmt.Printf("Saving config failed: %s\n", err)
		panic(1)
	}
	if ioutil.WriteFile(file, config, 0644) != nil {
		fmt.Printf("Saving config failed: %s\n", err)
		panic(1)
	}
}
