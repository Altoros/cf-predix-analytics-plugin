.PHONY: build

build:
	go build -o plugin *.go 
	cf uninstall-plugin PredixAnalyticsPlugin
	cf install-plugin -f plugin
	rm -f plugin

