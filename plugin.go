package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cloudfoundry/cli/cf/flags"
	"github.com/cloudfoundry/cli/cf/i18n"
	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/cloudfoundry/cli/cf/trace"
	"github.com/cloudfoundry/cli/plugin"
	go_i18n "github.com/nicksnyder/go-i18n/i18n"
	"gopkg.in/h2non/gentleman.v0"
)

type AnalyticsPlugin struct {
	ui            terminal.UI          `json:"-"`
	client        *gentleman.Client    `json:"-"`
	cliConnection plugin.CliConnection `json:"-"`
	UaaGuid       string               `json:"uaa_guid"`
	AnalyticsGuid string               `json:"analytics_guid"`
	AuthToken     string               `json:"auth_token"`
}

func main() {
	// T needs to point to a translate func, otherwise cf internals blow up
	tfunc, _ := go_i18n.Tfunc("")
	i18n.T = func(translationID string, args ...interface{}) string {
		return tfunc(translationID, args)
	}
	plugin.Start(new(AnalyticsPlugin))
}

func (p *AnalyticsPlugin) Run(cliConnection plugin.CliConnection, args []string) {
	if args[0] == "CLI-MESSAGE-UNINSTALL" {
		return
	}

	p.ui = terminal.NewUI(os.Stdin, os.Stdout, terminal.NewTeePrinter(os.Stdout), trace.NewWriterPrinter(os.Stdout, false))
	p.cliConnection = cliConnection

	p.loadConfig()
	p.checkLoggedIn()
	p.checkForPredix()
	p.checkForAnalyticsService()
	p.createClient()

	switch args[0] {
	case "taxonomy":
		p.getTaxonomy()
	case "add-taxonomy":
		if len(args) < 2 {
			fmt.Println("usage cf add-taxonomy <taxonomy>")
			panic(1)
		}
		p.addTaxonomy(args[1])
	case "analytics":
		p.listAnalytics()
	case "create-analytic":
		if len(args) < 3 {
			fmt.Println("usage cf create-analytic <Analytic name> <executable path>")
			panic(1)
		}
		var language string
		t := time.Now()
		defaultVersion := fmt.Sprintf("V1-%s", t.Format("Jan-2"))
		defaultAuthor, _ := cliConnection.Username()
		switch {
		case strings.HasSuffix(args[2], ".zip"):
			language = "Python"
		case strings.HasSuffix(args[2], ".jar"):
			language = "Java"
		}
		fc := flags.New()
		fc.NewStringFlagWithDefault("version", "v", "Analytic version", defaultVersion)
		fc.NewStringFlagWithDefault("author", "a", "Analytic author", defaultAuthor)
		fc.NewStringFlagWithDefault("language", "l", "Analytic supported language", language)
		fc.NewStringFlag("description", "d", "Analytic description")
		fc.NewStringFlag("taxonomy", "t", "Analytic taxonomy location")
		fc.NewStringFlag("metadata", "m", "Analytic custom metadata")
		err := fc.Parse(args[3:]...)
		if err != nil {
			fmt.Println(err)
			panic(1)
		}
		p.createAnalytic(args[1],
			args[2],
			fc.String("version"),
			fc.String("author"),
			fc.String("language"),
			fc.String("description"),
			fc.String("taxonomy"),
			fc.String("metadata"),
		)
	case "analytic-artifacts":
		if len(args) < 2 {
			fmt.Println("usage cf analytic-artifacts <Analytic name>")
			panic(1)
		}
		p.listArtifacts(args[1])
	case "get-analytic-artifact":
		if len(args) < 3 {
			fmt.Printf("usage cf get-analytic-artifact <Analytic name> <file name>\n", args[0])
			panic(1)
		}
		p.getArtifact(args[1], args[2])
	case "add-analytic-artifact":
		if len(args) < 3 {
			fmt.Printf("usage cf add-analytic-artifact <Analytic name> <file path> -type <artifact type> -description [artifact description]\n", args[0])
			panic(1)
		}
		fc := flags.New()
		fc.NewStringFlag("type", "t", "Artifact type")
		fc.NewStringFlag("descriptio", "d", "Artifact description")
		err := fc.Parse(args[3:]...)
		if err != nil {
			fmt.Println(err)
			panic(1)
		}
		if fc.IsSet("type") {
			p.addArtifact(args[1], args[2], fc.String("type"), fc.String("description"))
		} else {
			fmt.Println("Specify artifcat type")
			panic(1)
		}
	case "delete-analytic-artifact":
		if len(args) < 3 {
			fmt.Printf("usage cf delete-analytic-artifact <Analytic name> <file name>\n", args[0])
			panic(1)
		}
		p.deleteArtifact(args[1], args[2])
	case "run-analytic":
		if len(args) < 3 {
			fmt.Println("usage cf run-analytic <Analytic name> <input file>")
			panic(1)
		}
		p.runAnalytic(args[1], args[2])
	case "validate-analytic":
		if len(args) < 3 {
			fmt.Println("usage cf validate-analytic <Analytic name> <input file>")
			panic(1)
		}
		p.validateAnalytic(args[1], args[2])
	case "delete-analytic":
		if len(args) < 2 {
			fmt.Println("usage cf delete-analytic <Analytic name>")
			panic(1)
		}
		p.deleteAnalytic(args[1])
	case "analytic-logs":
		if len(args) < 2 {
			fmt.Println("usage cf analytic-logs <Analytic name>")
			panic(1)
		}
		p.analyticLogs(args[1])
	case "deploy-analytic":
		if len(args) < 2 {
			fmt.Println("usage cf deploy-analytic <Analytic name> [-memory mb] [-diskQuota mb] [-instances n]")
			panic(1)
		}
		fc := flags.New()
		fc.NewIntFlagWithDefault("memory", "m", "Memory size in MB", 512)
		fc.NewIntFlagWithDefault("diskQuota", "d", "Disk space in MB", 1024)
		fc.NewIntFlagWithDefault("instances", "i", "Number of instances", 1)
		err := fc.Parse(args[2:]...)
		if err != nil {
			fmt.Println(err)
			panic(1)
		}
		p.deployAnalytic(args[1], fc)
	case "analytics-curl":
		fs := make(map[string]flags.FlagSet)
		fs["i"] = &flags.BoolFlag{ShortName: "i", Usage: "Include response headers in the output"}
		fs["X"] = &flags.StringFlag{ShortName: "X", Usage: "HTTP method (GET,POST,PUT,DELETE,etc)"}
		fs["H"] = &flags.StringSliceFlag{ShortName: "H", Usage: "Custom headers to include in the request, flag can be specified multiple times"}
		fs["d"] = &flags.StringFlag{ShortName: "d", Usage: "HTTP data to include in the request body, or '@' followed by a file name to read the data from"}
		fs["output"] = &flags.StringFlag{Name: "output", Usage: "Write curl body to FILE instead of stdout"}
		ctx := flags.NewFlagContext(fs)
		err := ctx.Parse(args[1:]...)
		if err != nil {
			fmt.Println(err)
			panic(1)
		}
		p.curl(ctx)
	}

}

func (c *AnalyticsPlugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "PredixAnalyticsPlugin",
		Version: plugin.VersionType{
			Major: 1,
			Minor: 0,
			Build: 0,
		},
		MinCliVersion: plugin.VersionType{
			Major: 6,
			Minor: 7,
			Build: 0,
		},
		Commands: []plugin.Command{
			{
				Name:     "analytics",
				HelpText: "List analytics",

				UsageDetails: plugin.Usage{
					Usage: "analytics\n   cf analytics",
				},
			},
			{
				Name:     "create-analytic",
				HelpText: "Create analytic",

				UsageDetails: plugin.Usage{
					Usage: "create-analytic\n  cf create-analytic <Analytic name> <path to executable> [-version version] [-author] [-description description] [-taxonomy taxonomy location] [-language (Python|Java|Matlab)] [-metadata custom analytic metadata]",
				},
			},
			{
				Name:     "delete-analytic",
				HelpText: "Delete analytic",

				UsageDetails: plugin.Usage{
					Usage: "delete-analytic\n   cf delete-analytic <Analytic name>",
				},
			},
			{
				Name:     "validate-analytic",
				HelpText: "Validate analytic",

				UsageDetails: plugin.Usage{
					Usage: "validate-analytic\n   cf validate-analytic <Analytic name> <input file>",
				},
			},
			{
				Name:     "deploy-analytic",
				HelpText: "deploy analytic",

				UsageDetails: plugin.Usage{
					Usage: "deploy-analytic\n   cf deploy-analytic <Analytic name>",
				},
			},
			{
				Name:     "run-analytic",
				HelpText: "Run analytic",

				UsageDetails: plugin.Usage{
					Usage: "run-analytic\n   cf run-analytic <Analytic name> <input file>",
				},
			},
			{
				Name:     "analytic-logs",
				HelpText: "Get the recent analytic logs",

				UsageDetails: plugin.Usage{
					Usage: "analytic-logs\n   cf analytic-logs <Analytic name>",
				},
			},
			{
				Name:     "analytic-artifacts",
				HelpText: "List analytic artifacts",

				UsageDetails: plugin.Usage{
					Usage: "analytic-artifacts\n   cf analytic-artifacts <Analytic name>",
				},
			},
			{
				Name:     "get-analytic-artifact",
				HelpText: "Get analytic artifact",

				UsageDetails: plugin.Usage{
					Usage: "get-analytic-artifacts\n   cf get-analytic-artifact <Analytic name> <file name>",
				},
			},
			{
				Name:     "add-analytic-artifact",
				HelpText: "Add analytic artifact",

				UsageDetails: plugin.Usage{
					Usage: "add-analytic-artifacts\n   cf add-analytic-artifact <Analytic name> <file name> -type <artifact type> -description [artifact description]",
				},
			},
			{
				Name:     "delete-analytic-artifact",
				HelpText: "Delete analytic artifact",

				UsageDetails: plugin.Usage{
					Usage: "delete-analytic-artifacts\n   cf delete-analytic-artifact <Analytic name> <file name>",
				},
			},
			{
				Name:     "taxonomy",
				HelpText: "Retrieves the full taxonomy structure",

				UsageDetails: plugin.Usage{
					Usage: "taxonomy\n   cf taxonomy",
				},
			},
			{
				Name:     "add-taxonomy",
				HelpText: "Add taxonomy into catalog",

				UsageDetails: plugin.Usage{
					Usage: "add-taxonomy\n   cf add-taxonomy <Taxonomy>",
				},
			},
			{
				Name:     "analytics-curl",
				HelpText: "Executes a request to the targeted Analytics Catalog API endpoint",

				UsageDetails: plugin.Usage{
					Usage: "analytics-curl\n  cf analytics-curl PATH [-iv] [-X METHOD] [-H HEADER] [-d DATA] [--output FILE]",
				},
			},
		},
	}
}
