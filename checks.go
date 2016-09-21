package main

import (
	"fmt"
	"strings"
)

func (p *AnalyticsPlugin) checkForPredix() {
	hasEndpoint, _ := p.cliConnection.HasAPIEndpoint()
	if !hasEndpoint {
		fmt.Println("API endpoint not specified")
		panic(1)
	}
	apiEndpoint, e := p.cliConnection.ApiEndpoint()
	if e != nil || !strings.HasSuffix(apiEndpoint, "predix.io") {
		fmt.Println("Not predix!")
		panic(1)
	}
}

func (p *AnalyticsPlugin) checkLoggedIn() {
	cliLogged, err := p.cliConnection.IsLoggedIn()
	if err != nil {
		p.ui.Failed(err.Error())
	}

	if cliLogged == false {
		panic("cannot manage analytics without being logged in to CF")
	}
}

func (p *AnalyticsPlugin) checkForAnalyticsService() {
	p.analyticsServiceGuid()
}
