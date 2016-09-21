package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/cli/cf/flags"
	"github.com/cloudfoundry/cli/cf/util"
)

func (p *AnalyticsPlugin) curl(c flags.FlagContext) {
	path := c.Args()[0]
	headers := make(map[string]string)

	for _, header := range c.StringSlice("H") {
		h := strings.SplitN(header, ":", 2)
		if len(h) > 1 {
			headers[h[0]] = strings.Trim(h[1], " ")
		}
	}

	var method string
	var body string

	if c.IsSet("d") {
		method = "POST"

		jsonBytes, err := util.GetContentsFromOptionalFlagValue(c.String("d"))
		if err != nil {
			fmt.Printf("Error creating request: %s\n", err)
			panic(1)
		}
		body = string(jsonBytes)
	}

	if c.IsSet("X") {
		method = c.String("X")
	}

	if method == "" && body != "" {
		method = "POST"
	}
	req := p.client.Request().Method(method).Path("/api").AddPath(path).SetHeaders(headers)
	if body != "" {
		req = req.BodyString(body)
	}
	resp, err := req.Do()
	if err != nil {
		fmt.Printf("Error creating request: %s\n", err)
		panic(1)
	}

	responseBody := resp.String()

	if c.Bool("i") {
		responseHeader := bytes.Buffer{}
		resp.Header.Write(&responseHeader)
		p.ui.Say(responseHeader.String())
	}
	if c.String("output") != "" {
		err = writeToFile(responseBody, c.String("output"))
		if err != nil {
			fmt.Printf("Error creating request: %s\n", err)
			panic(1)
		}
	} else {
		if resp.Header.Get("Content-Type") == "application/json" {
			buffer := bytes.Buffer{}
			err := json.Indent(&buffer, []byte(responseBody), "", "   ")
			if err == nil {
				responseBody = buffer.String()
			}
		}

		p.ui.Say(responseBody)
	}

}

func writeToFile(responseBody, filePath string) (err error) {
	if _, err = os.Stat(filePath); os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(filePath), 0755)
	}

	if err != nil {
		return
	}

	return ioutil.WriteFile(filePath, []byte(responseBody), 0644)
}
