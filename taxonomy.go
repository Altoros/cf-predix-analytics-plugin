package main

import (
	"fmt"
	"strings"
)

type Taxonomy struct {
	Name   string     `json:"node_name"`
	Childs []Taxonomy `json:"child_nodes"`
}

func printTaxonomy(parent string, taxonomy Taxonomy) {
	p := fmt.Sprintf("%s/%s", parent, taxonomy.Name)
	fmt.Println(p)
	for _, t := range taxonomy.Childs {
		printTaxonomy(p, t)
	}

}

func (p *AnalyticsPlugin) getTaxonomy() {
	r, e := p.client.Get().Path("/api/v1/catalog/taxonomy").Do()
	if e != nil {
		fmt.Printf("Failed to get taxonomy: %s\n", e)
		panic(1)
	}
	var taxonomy []Taxonomy
	r.JSON(&taxonomy)
	for _, t := range taxonomy {
		printTaxonomy("", t)
	}
}

func (p *AnalyticsPlugin) addTaxonomy(t string) {
	taxonomies := strings.Split(t, "/")
	var taxonomy Taxonomy
	tt := &taxonomy
	for _, t := range taxonomies {
		if t != "" {
			tt.Name = t
			tt.Childs = make([]Taxonomy, 1)
			tt = &(tt.Childs[0])
		}
	}
	fmt.Printf("Adding `%s` taxonomy...\n", t)
	_, e := p.client.Post().Path("/api/v1/catalog/taxonomy").JSON(taxonomy).Do()
	if e != nil {
		fmt.Printf("Failed to add taxonomy: %s\n", e)
		panic(1)
	}
	p.ui.Ok()
}
