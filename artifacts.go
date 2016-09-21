package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

type ArtifactList struct {
	Artifacts []Artifact `json:"artifacts"`
}

type Artifact struct {
	Id               string `json:"id"`
	Filename         string `json:"filename"`
	Type             string `json:"type"`
	Description      string `json:"description"`
	CreatedTimestamp string `json:"createdTimestamp"`
	UpdatedTimestamp string `json:"updatedTimestamp"`
}

func (p *AnalyticsPlugin) artifactsList(analyticName string) []Artifact {
	r, e := p.client.Get().Path(fmt.Sprintf("/api/v1/catalog/analytics/%s/artifacts", p.analyticId(analyticName))).Do()
	if e != nil {
		fmt.Printf("Failed to get analytic artifacts: %s\n", e)
		panic(1)
	}
	var artifacts ArtifactList
	r.JSON(&artifacts)
	return artifacts.Artifacts
}

func (p *AnalyticsPlugin) artifactId(analyticName, artifactName string) string {
	artifactId := ""
	for _, artifact := range p.artifactsList(analyticName) {
		if artifact.Filename == artifactName {
			artifactId = artifact.Id
			break
		}
	}
	if artifactId == "" {
		fmt.Printf("Artifact %s not found\n", artifactName)
		panic(1)
	}
	return artifactId
}

func (p *AnalyticsPlugin) artifactUrl(analyticName, artifactName string) string {
	return fmt.Sprintf("/api/v1/catalog/artifacts/%s/file", p.artifactId(analyticName, artifactName))
}

func (p *AnalyticsPlugin) getArtifact(analyticName, artifactName string) {
	r, e := p.client.Get().Path(p.artifactUrl(analyticName, artifactName)).Do()
	if e != nil {
		fmt.Printf("Failed to get artifact: %s\n", e)
		panic(1)
	}
	if r.StatusCode == 200 {
		r.SaveToFile(artifactName)
	} else {
		fmt.Println(r.String())
	}
}
func (p *AnalyticsPlugin) deleteArtifact(analyticName, artifactName string) {
	r, e := p.client.Delete().Path(p.artifactUrl(analyticName, artifactName)).Do()
	if e != nil {
		fmt.Printf("Failed to delete artifact: %s\n", e)
		panic(1)
	}
	if r.StatusCode == 204 {
		fmt.Println("The artifact was removed from the catalog.")
	} else {
		fmt.Printf("Failed to delete artifact. Response status code: %d\n", r.StatusCode)
	}
}

func (p *AnalyticsPlugin) listArtifacts(analyticName string) {
	table := p.ui.Table([]string{"Filename", "Type", "Description"})
	for _, artifact := range p.artifactsList(analyticName) {
		table.Add(artifact.Filename, artifact.Type, artifact.Description)
	}
	table.Print()
}

func (p *AnalyticsPlugin) addArtifact(analyticName, artifactPath, artifactType, description string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("catalogEntryId", p.analyticId(analyticName))
	writer.WriteField("type", artifactType)
	if description != "" {
		writer.WriteField("description", description)
	}
	file, err := os.Open(artifactPath)
	if err != nil {
		fmt.Printf("Failed to upload artifact: %s\n", err)
		panic(1)
	}
	part, err := writer.CreateFormFile("file", filepath.Base(artifactPath))
	if err != nil {
		fmt.Printf("Failed to upload artifact: %s\n", err)
		panic(1)
	}
	_, err = io.Copy(part, file)
	writer.Close()
	_, e := p.client.Post().Path("/api/v1/catalog/artifacts").Body(body).SetHeader("Content-Type", writer.FormDataContentType()).Do()
	if e != nil {
		fmt.Printf("Failed to upload artifact: %s\n", e)
		panic(1)
	}
}
