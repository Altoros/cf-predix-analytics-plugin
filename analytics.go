package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/cloudfoundry/cli/cf/flags"
)

type AnalyticCatalogEntry struct {
	Id                string `json:"id,omitempty"`
	Name              string `json:"name,omitempty"`
	Author            string `json:"author,omitempty"`
	Description       string `json:"description,omitempty"`
	Version           string `json:"version,omitempty"`
	SupportedLanguage string `json:"supportedLanguage,omitempty"`
	CustomMetadata    string `json:"customMetadata,omitempty"`
	TaxonomyLocation  string `json:"taxonomyLocation,omitempty"`
	State             string `json:"state,omitempty"`
	CreatedTimestamp  string `json:"createdTimestamp,omitempty"`
	UpdatedTimestamp  string `json:"updatedTimestamp,omitempty"`
}

type AnalyticCatalogEntryPage struct {
	CurrentPageSize   int                    `json:"currentPageSize"`
	MaximumPageSize   int                    `json:"maximumPageSize"`
	CurrentPageNumber int                    `json:"currentPageNumber"`
	TotalElements     int                    `json:"totalElements"`
	TotalPages        int                    `json:"totalPages"`
	Entries           []AnalyticCatalogEntry `json:"analyticCatalogEntries"`
}

type AnalyticValidationResult struct {
	AnalyticId          string `json:"analyticId"`
	ValidationRequestId string `json:"validationRequestId"`
	Status              string `json:"status"`
	Message             string `json:"message"`
	InputData           string `json:"inputData"`
	Result              string `json:"result"`
	CreatedTimestamp    string `json:"createdTimestamp"`
	UpdatedTimestamp    string `json:"updatedTimestamp"`
}

type AnalyticDeploymentResult struct {
	AnalyticId       string `json:"analyticId"`
	RequestId        string `json:"requestId"`
	Status           string `json:"status"`
	Message          string `json:"message"`
	InputConfigData  string `json:"inputConfigData"`
	Result           string `json:"result"`
	CreatedTimestamp string `json:"createdTimestamp"`
	UpdatedTimestamp string `json:"updatedTimestamp"`
}

type AnalyticDeploymentConfiguration struct {
	Memory    int `json:"memory"`
	DiskQuota int `json:"diskQuota"`
	Instances int `json:"instances"`
}

func (p *AnalyticsPlugin) analyticsList() []AnalyticCatalogEntry {
	r, e := p.client.Get().Path("/api/v1/catalog/analytics").Do()
	if e != nil {
		fmt.Printf("Failed to get analytics list: %s\n", e)
		panic(1)
	}
	var analytics AnalyticCatalogEntryPage
	r.JSON(&analytics)
	return analytics.Entries
}

func (p *AnalyticsPlugin) analyticId(analyticName string) string {
	analyticId := ""
	for _, analytic := range p.analyticsList() {
		if analytic.Name == analyticName {
			analyticId = analytic.Id
			break
		}
	}
	if analyticId == "" {
		fmt.Printf("Analytic %s not found\n", analyticName)
		panic(1)
	}
	return analyticId
}

func (p *AnalyticsPlugin) listAnalytics() {
	p.ui.Say("Getting analytics list...")
	analytics := p.analyticsList()
	p.ui.Ok()
	table := p.ui.Table([]string{"Name", "Version", "Taxonomy Location", "Author", "Description"})
	for _, analytic := range analytics {
		table.Add(analytic.Name,
			analytic.Version,
			analytic.TaxonomyLocation,
			analytic.Author,
			analytic.Description,
		)
	}
	table.Print()
}

func (p *AnalyticsPlugin) runAnalytic(analyticName, inputFilePath string) {
	input, err := os.Open(inputFilePath)
	if err != nil {
		fmt.Printf("Error accessing input file: %s\n", err)
		panic(1)
	}
	req := p.client.Post().Path(fmt.Sprintf("/api/v1/catalog/analytics/%s/execution", p.analyticId(analyticName)))
	req = req.Body(input)
	r, e := req.Do()
	if e != nil {
		fmt.Printf("Failed to run analytic: %s\n", e)
		panic(1)
	}
	fmt.Println(r.String())
}

func (p *AnalyticsPlugin) validateAnalytic(analyticName, inputFilePath string) {
	input, err := os.Open(inputFilePath)
	if err != nil {
		fmt.Printf("Error accessing input file: %s\n", err)
		panic(1)
	}
	analyticId := p.analyticId(analyticName)
	req := p.client.Post().Path(fmt.Sprintf("/api/v1/catalog/analytics/%s/validation", analyticId))
	req = req.Body(input)
	r, e := req.Do()
	if e != nil {
		fmt.Printf("Failed to validate analytic: %s\n", e)
		panic(1)
	}
	var result AnalyticValidationResult
	e = r.JSON(&result)
	validateUrl := fmt.Sprintf("/api/v1/catalog/analytics/%s/validation/%s", analyticId, result.ValidationRequestId)
	r, e = p.client.Get().Path(validateUrl).Do()
	for e == nil {
		e = r.JSON(&result)
		if result.Status == "COMPLETED" || result.Status == "ERROR" {
			break
		}
		r, e = p.client.Get().Path(validateUrl).Do()
	}
	if e != nil {
		fmt.Printf("Failed to validate analytic: %s\n", e)
		panic(1)
	}
	p.ui.Say(result.Message)
}

func (p *AnalyticsPlugin) deleteAnalytic(analyticName string) {
	_, e := p.client.Delete().Path(fmt.Sprintf("/api/v1/catalog/analytics/%s", p.analyticId(analyticName))).Do()
	if e != nil {
		fmt.Printf("Failed to delete analytic: %s\n", e)
		panic(1)
	}
}

func (p *AnalyticsPlugin) createAnalytic(name, executablePath, version, author, language, description, taxonomyLocation, metadata string) {
	analytic := AnalyticCatalogEntry{
		Name:              name,
		Author:            author,
		Description:       description,
		Version:           version,
		SupportedLanguage: language,
		TaxonomyLocation:  taxonomyLocation,
		CustomMetadata:    metadata,
	}
	_, e := p.client.Post().Path("/api/v1/catalog/analytics").JSON(analytic).Do()
	if e != nil {
		fmt.Printf("Failed to create analytic: %s\n", e)
		panic(1)
	}
	p.addArtifact(name, executablePath, "Executable", "")
}

func (p *AnalyticsPlugin) analyticLogs(name string) {
	r, e := p.client.Get().Path(fmt.Sprintf("/api/v1/catalog/analytics/%s/logs", p.analyticId(name))).Do()
	if e != nil {
		fmt.Printf("Failed to get analytic logs: %s\n", e)
		panic(1)
	}
	p.ui.Say(strings.Replace(r.String(), "(STD", "\r(STD", -1))
}

func (p *AnalyticsPlugin) deployAnalytic(name string, c flags.FlagContext) {
	config := AnalyticDeploymentConfiguration{
		c.Int("memory"),
		c.Int("diskQuota"),
		c.Int("instances"),
	}
	analyticId := p.analyticId(name)
	r, e := p.client.Post().Path(fmt.Sprintf("/api/v1/catalog/analytics/%s/deployment", analyticId)).JSON(config).Do()
	if e != nil {
		fmt.Printf("Failed to deploy analytic: %s\n", e)
		panic(1)
	}
	var result AnalyticDeploymentResult
	e = r.JSON(&result)
	if e != nil {
		fmt.Printf("Failed to deploy analytic: %s\n", e)
		panic(1)
	}
	statusUrl := fmt.Sprintf("/api/v1/catalog/analytics/%s/deployment/%s", analyticId, result.RequestId)
	r, e = p.client.Get().Path(statusUrl).Do()
	for e == nil {
		e = r.JSON(&result)
		if result.Status == "COMPLETED" || result.Status == "ERROR" {
			break
		}
		r, e = p.client.Get().Path(statusUrl).Do()
	}
	if e != nil {
		fmt.Printf("Failed to deploy analytic: %s\n", e)
		panic(1)
	}
	p.ui.Say(result.Message)
}
