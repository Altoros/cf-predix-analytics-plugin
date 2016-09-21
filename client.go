package main

import (
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"gopkg.in/h2non/gentleman.v0"
)

func (p *AnalyticsPlugin) uaaServiceGuid() string {
	if p.UaaGuid == "" {
		services, e := p.cliConnection.GetServices()
		if e != nil {
			fmt.Printf("Failed to get services: %s", e)
			panic(1)
		}
		for _, s := range services {
			if s.Service.Name == "predix-uaa" {
				service, _ := p.cliConnection.GetService(s.Name)
				p.UaaGuid = service.Guid
				return p.UaaGuid
			}
		}
		fmt.Println("UAA service not found")
		panic(1)
	}
	return p.UaaGuid
}

func (p *AnalyticsPlugin) analyticsServiceGuid() string {
	if p.AnalyticsGuid == "" {
		services, e := p.cliConnection.GetServices()
		if e != nil {
			fmt.Printf("Failed to get services: %s", e)
			panic(1)
		}
		for _, s := range services {
			if s.Service.Name == "predix-analytics-catalog" {
				service, _ := p.cliConnection.GetService(s.Name)
				p.AnalyticsGuid = service.Guid
				return p.AnalyticsGuid
			}
		}
		fmt.Println("Analytics service not found")
		panic(1)
	}
	return p.AnalyticsGuid
}

func (p *AnalyticsPlugin) authToken() string {
	if p.AuthToken == "" {
		clientID := p.ui.Ask("Client ID")
		clientSecret := p.ui.AskForPassword("Client secret")
		conf := clientcredentials.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			TokenURL:     fmt.Sprintf("https://%s.predix-uaa.run.aws-usw02-pr.ice.predix.io/oauth/token", p.uaaServiceGuid()),
		}
		t, err := conf.Token(oauth2.NoContext)
		if err != nil {
			fmt.Printf("Auth failed: %s\n", err)
			panic(1)
		}
		p.AuthToken = t.AccessToken
	}
	return p.AuthToken
}

func (p *AnalyticsPlugin) invalidAuth() bool {
	r, e := p.client.Get().Path("/api/v1/catalog/taxonomy").Do()
	return e != nil || r.StatusCode != 200
}

func (p *AnalyticsPlugin) createClient() {
	p.client = gentleman.New()
	p.client.BaseURL("https://predix-analytics-catalog-release.run.aws-usw02-pr.ice.predix.io")
	p.client.SetHeader("Authorization", fmt.Sprintf("Bearer %s", p.authToken()))
	p.client.SetHeader("Predix-Zone-Id", p.analyticsServiceGuid())
	for p.invalidAuth() {
		p.AuthToken = ""
		p.client.SetHeader("Authorization", fmt.Sprintf("Bearer %s", p.authToken()))
	}
	p.saveConfig()
}
