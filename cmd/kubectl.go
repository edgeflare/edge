package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/edgeflare/edge/pkg/kube"
	"github.com/urfave/cli/v2"
	"sigs.k8s.io/yaml"
)

func kubectlCommand() *cli.Command {
	return &cli.Command{
		Name:    "kubectl",
		Aliases: []string{"k"},
		Usage:   "run kubectl-like commands",
		Subcommands: []*cli.Command{
			{
				Name:      "get",
				Aliases:   []string{"g"},
				Usage:     "Get resources. Supply resource arg in PLURAL, eg, pods, namespaces, deployments",
				Action:    getResource,
				ArgsUsage: "[resource] [--namespace namespace]",
			},
			{
				Name:    "apply",
				Aliases: []string{"create", "a"},
				Usage:   "Create a resource, or patch/update if it already exists",
				Action: func(c *cli.Context) error {
					var fileBytes []byte
					var err error

					if c.String("file") == "-" {
						// Read from stdin
						fileBytes, err = io.ReadAll(os.Stdin)
						if err != nil {
							fmt.Println("Error reading from stdin:", err)
							return err
						}
					} else {
						// Read from file
						fileBytes, err = os.ReadFile(c.String("file"))
						if err != nil {
							fmt.Println("Error reading file:", err)
							return err
						}
					}

					if err := kube.ApplyResource(fileBytes); err != nil {
						fmt.Println("Error applying resource:", err)
						return err
					}

					fmt.Println("Applied resource successfully")
					return nil
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "file",
						Aliases:  []string{"f"},
						Usage:    "Filename in JSON or YAML. Use '-' to read from stdin.",
						Required: true,
					},
				},
			},
			{
				Name:    "delete",
				Aliases: []string{"d"},
				Usage:   "Delete a resource",
				Action: func(c *cli.Context) error {
					var fileContent []byte
					var err error

					if c.IsSet("file") {
						fileContent, err = os.ReadFile(c.String("file"))
						if err != nil {
							fmt.Println(err)
							return err
						}
					}

					resourceType := c.Args().Get(0)
					resourceName := c.Args().Get(1)
					namespace := c.String("namespace")

					if err := kube.DeleteResource(resourceType, resourceName, namespace, fileContent); err != nil {
						fmt.Println(err)
						return err
					}

					fmt.Println("Resource deleted successfully")
					return nil
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "file",
						Aliases: []string{"f"},
						Usage:   "Filename, directory, or URL to files to use to delete the resource",
					},
				},
				ArgsUsage: "[resourceType resourceName]",
			},
		},
		Flags: kubectlFlags,
	}
}

var kubectlFlags = []cli.Flag{
	&cli.StringFlag{
		Name:    "namespace",
		Aliases: []string{"n"},
		Usage:   "Namespace to use for the kubectl command",
		Value:   "",
	},
	&cli.StringFlag{
		Name:    "output",
		Aliases: []string{"o"},
		Value:   "name",
		Usage:   "Output format. One of: name|yaml|json",
	},
}

func getResource(c *cli.Context) error {
	if c.NArg() == 0 {
		return fmt.Errorf("you must specify the resource type")
	}

	resourceType := c.Args().Get(0)
	var resourceName string
	if c.NArg() > 1 {
		resourceName = c.Args().Get(1) // Get the specific resource name if provided
	}

	resources, err := kube.GetResources(resourceType, resourceName, c.String("namespace"))
	if err != nil {
		return fmt.Errorf("error getting Kubernetes resources: %v", err)
	}

	outputFormat := c.String("output")
	isSingleResource := resourceName != "" && len(resources) == 1

	switch outputFormat {
	case "yaml":
		var yamlBytes []byte
		if isSingleResource {
			yamlBytes, err = yaml.Marshal(resources[0].Object)
		} else {
			yamlBytes, err = yaml.Marshal(resources)
		}
		if err != nil {
			return fmt.Errorf("error marshaling to YAML: %v", err)
		}
		fmt.Printf("%s", string(yamlBytes))

	case "json":
		var jsonBytes []byte
		if isSingleResource {
			jsonBytes, err = json.MarshalIndent(resources[0].Object, "", "  ")
		} else {
			jsonBytes, err = json.MarshalIndent(resources, "", "  ")
		}
		if err != nil {
			return fmt.Errorf("error marshaling to JSON: %v", err)
		}
		fmt.Printf("%s\n", string(jsonBytes))

	case "name", "": // default to name if not specified
		if isSingleResource {
			fmt.Printf("%-30s %-30s\n", "NAME", "NAMESPACE") // Header
			fmt.Printf("%-30s %-30s\n", resources[0].GetName(), resources[0].GetNamespace())
		} else {
			fmt.Printf("%-30s %-30s\n", "NAME", "NAMESPACE") // Header
			for _, r := range resources {
				fmt.Printf("%-30s %-30s\n", r.GetName(), r.GetNamespace())
			}
		}

	default:
		return fmt.Errorf("invalid output format: %s", outputFormat)
	}

	return nil
}
