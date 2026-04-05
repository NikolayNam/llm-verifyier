package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	llmprovider "github.com/NikolayNam/collabsphere/platform-tooling/internal/providermanifest"
)

func main() {
	if len(os.Args) < 2 {
		fail("usage: provider-manifest <list|render|activate> [flags]")
	}

	catalogPath, templatePath, activatedPath, err := llmprovider.DefaultPaths()
	if err != nil {
		fail(err.Error())
	}

	switch os.Args[1] {
	case "list":
		listCmd := flag.NewFlagSet("list", flag.ExitOnError)
		catalogFile := listCmd.String("catalog", catalogPath, "Path to llm_provider catalog.")
		_ = listCmd.Parse(os.Args[2:])
		catalog, err := llmprovider.LoadCatalog(*catalogFile)
		if err != nil {
			fail(err.Error())
		}
		for _, model := range catalog.Models {
			status := "inactive"
			if model.Active {
				status = "active"
			}
			fmt.Printf("%s\t%s\n", status, model.Ref)
		}
	case "render":
		renderCmd := flag.NewFlagSet("render", flag.ExitOnError)
		catalogFile := renderCmd.String("catalog", catalogPath, "Path to llm_provider catalog.")
		templateFile := renderCmd.String("template", templatePath, "Path to llm_provider template.")
		outputFile := renderCmd.String("output", activatedPath, "Path to generated activated compose.")
		_ = renderCmd.Parse(os.Args[2:])
		catalog, err := llmprovider.LoadCatalog(*catalogFile)
		if err != nil {
			fail(err.Error())
		}
		template, err := llmprovider.LoadTemplate(*templateFile)
		if err != nil {
			fail(err.Error())
		}
		rendered, err := llmprovider.RenderActivated(template, catalog)
		if err != nil {
			fail(err.Error())
		}
		if err := llmprovider.SaveActivated(*outputFile, rendered); err != nil {
			fail(err.Error())
		}
		fmt.Printf("rendered %s with %d active model(s)\n", *outputFile, len(llmprovider.ActiveRefs(catalog)))
	case "activate":
		activateCmd := flag.NewFlagSet("activate", flag.ExitOnError)
		catalogFile := activateCmd.String("catalog", catalogPath, "Path to llm_provider catalog.")
		templateFile := activateCmd.String("template", templatePath, "Path to llm_provider template.")
		outputFile := activateCmd.String("output", activatedPath, "Path to generated activated compose.")
		refsValue := activateCmd.String("refs", "", "Comma-separated model refs to activate.")
		_ = activateCmd.Parse(os.Args[2:])
		refs := parseRefs(*refsValue)
		if len(refs) == 0 {
			fail("activate requires --refs with at least one model ref")
		}
		catalog, err := llmprovider.LoadCatalog(*catalogFile)
		if err != nil {
			fail(err.Error())
		}
		if err := llmprovider.ActivateRefs(catalog, refs); err != nil {
			fail(err.Error())
		}
		if err := llmprovider.SaveCatalog(*catalogFile, catalog); err != nil {
			fail(err.Error())
		}
		template, err := llmprovider.LoadTemplate(*templateFile)
		if err != nil {
			fail(err.Error())
		}
		rendered, err := llmprovider.RenderActivated(template, catalog)
		if err != nil {
			fail(err.Error())
		}
		if err := llmprovider.SaveActivated(*outputFile, rendered); err != nil {
			fail(err.Error())
		}
		fmt.Printf("activated %d model(s) and rendered %s\n", len(refs), *outputFile)
	default:
		fail("unknown subcommand: " + os.Args[1])
	}
}

func parseRefs(value string) []string {
	parts := strings.Split(value, ",")
	refs := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		refs = append(refs, part)
	}
	return refs
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
