package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// Docker Catalog structures (input)
type DockerCatalogEntry struct {
	Description string                 `json:"description"`
	Title       string                 `json:"title"`
	Type        string                 `json:"type"`
	DateAdded   string                 `json:"dateAdded"`
	Image       string                 `json:"image"`
	Ref         string                 `json:"ref"`
	Readme      string                 `json:"readme"`
	ToolsURL    string                 `json:"toolsUrl"`
	Source      string                 `json:"source"`
	Upstream    string                 `json:"upstream"`
	Icon        string                 `json:"icon"`
	Tools       []Tool                 `json:"tools"`
	Prompts     int                    `json:"prompts"`
	Resources   map[string]interface{} `json:"resources"`
	Metadata    Metadata               `json:"metadata"`
	Secrets     []Secret               `json:"secrets,omitempty"`
	Env         []EnvVar               `json:"env,omitempty"`
	Command     []string               `json:"command,omitempty"`
	Volumes     []string               `json:"volumes,omitempty"`
	Config      []Config               `json:"config,omitempty"`
	Remote      *Remote                `json:"remote,omitempty"`
	User        string                 `json:"user,omitempty"`
	LongLived   bool                   `json:"longLived,omitempty"`
	AllowHosts  []string               `json:"allowHosts,omitempty"`
	OAuth       *OAuth                 `json:"oauth,omitempty"`
}

type Tool struct {
	Name string `json:"name"`
}

type Metadata struct {
	Pulls       int      `json:"pulls,omitempty"`
	Stars       int      `json:"stars,omitempty"`
	GitHubStars int      `json:"githubStars"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	License     string   `json:"license"`
	Owner       string   `json:"owner"`
}

type Secret struct {
	Name    string `json:"name"`
	Env     string `json:"env"`
	Example string `json:"example"`
}

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Config struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Properties  map[string]interface{} `json:"properties"`
	Required    []string               `json:"required,omitempty"`
}

type Remote struct {
	TransportType string            `json:"transport_type"`
	URL           string            `json:"url"`
	Headers       map[string]string `json:"headers,omitempty"`
}

type OAuth struct {
	Providers []OAuthProvider `json:"providers"`
}

type OAuthProvider struct {
	Provider string `json:"provider"`
	Secret   string `json:"secret"`
	Env      string `json:"env"`
}

type DockerCatalog struct {
	Name        string                        `json:"name"`
	DisplayName string                        `json:"displayName"`
	Registry    map[string]DockerCatalogEntry `json:"registry"`
}

// Registry Schema structures (output)
type RegistryServer struct {
	Schema      string                 `json:"$schema"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	Packages    []Package              `json:"packages,omitempty"`
	Remotes     []Transport            `json:"remotes,omitempty"`
	Repository  *Repository            `json:"repository,omitempty"`
	WebsiteURL  string                 `json:"websiteUrl,omitempty"`
	Meta        map[string]interface{} `json:"_meta,omitempty"`
}

type Package struct {
	RegistryType         string          `json:"registryType"`
	Transport            Transport       `json:"transport,omitempty"`
	Identifier           string          `json:"identifier"`
	Version              string          `json:"version,omitempty"`
	EnvironmentVariables []KeyValueInput `json:"environmentVariables,omitempty"`
	PackageArguments     []Argument      `json:"packageArguments,omitempty"`
	RuntimeArguments     []Argument      `json:"runtimeArguments,omitempty"`
	RuntimeHint          string          `json:"runtimeHint,omitempty"`
}

type Transport struct {
	Type    string          `json:"type"`
	URL     string          `json:"url,omitempty"`
	Headers []KeyValueInput `json:"headers,omitempty"`
}

type KeyValueInput struct {
	Name        string                 `json:"name"`
	Value       string                 `json:"value,omitempty"`
	Description string                 `json:"description,omitempty"`
	IsRequired  bool                   `json:"isRequired,omitempty"`
	IsSecret    bool                   `json:"isSecret,omitempty"`
	IsRepeated  bool                   `json:"isRepeated,omitempty"`
	Variables   map[string]InputSchema `json:"variables,omitempty"`
}

type Argument struct {
	Type       string                 `json:"type"`
	Name       string                 `json:"name,omitempty"`
	Value      string                 `json:"value,omitempty"`
	IsRequired bool                   `json:"isRequired,omitempty"`
	IsSecret   bool                   `json:"isSecret,omitempty"`
	IsRepeated bool                   `json:"isRepeated,omitempty"`
	Variables  map[string]InputSchema `json:"variables,omitempty"`
}

type InputSchema struct {
	IsSecret    bool   `json:"isSecret,omitempty"`
	IsRequired  bool   `json:"isRequired,omitempty"`
	Format      string `json:"format,omitempty"`
	Description string `json:"description,omitempty"`
}

type Repository struct {
	URL       string `json:"url"`
	Source    string `json:"source,omitempty"`
	Subfolder string `json:"subfolder,omitempty"`
}

func main() {
	// Run docker mcp catalog show --format json
	cmd := exec.Command("docker", "mcp", "catalog", "show", "--format", "json")
	data, err := cmd.Output()
	if err != nil {
		log.Fatalf("Error running docker mcp catalog show: %v", err)
	}

	var catalog DockerCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		log.Fatalf("Error parsing catalog JSON: %v", err)
	}

	servers := make([]RegistryServer, 0, len(catalog.Registry))
	for name, entry := range catalog.Registry {
		// Skip remote servers (they have namespace validation issues)
		if entry.Type == "remote" {
			continue
		}

		server := transformEntry(name, entry)
		servers = append(servers, server)
	}

	outputData, err := json.MarshalIndent(servers, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling output: %v", err)
	}

	if err := os.WriteFile("seed.json", outputData, 0644); err != nil {
		log.Fatalf("Error writing seed.json: %v", err)
	}

	fmt.Println("Successfully created seed.json")
}

func buildConfigMap(configs []Config) map[string]map[string]interface{} {
	result := make(map[string]map[string]interface{})

	for _, cfg := range configs {
		for key, val := range cfg.Properties {
			fullKey := cfg.Name + "." + key
			if propMap, ok := val.(map[string]interface{}); ok {
				result[fullKey] = propMap
			}
		}
	}

	return result
}

func transformEntry(name string, entry DockerCatalogEntry) RegistryServer {
	// Truncate description to 100 characters max
	description := entry.Description
	if len(description) > 100 {
		description = description[:97] + "..."
	}

	server := RegistryServer{
		Schema:      "https://static.modelcontextprotocol.io/schemas/2025-10-17/server.schema.json",
		Name:        fmt.Sprintf("com.docker.mcp/%s", name),
		Description: description,
		Version:     "v0.1.0",
		Meta:        make(map[string]interface{}),
	}

	configMap := buildConfigMap(entry.Config)

	publisherProvided := map[string]interface{}{
		"pulls":       entry.Metadata.Pulls,
		"githubStars": entry.Metadata.GitHubStars,
		"category":    entry.Metadata.Category,
		"tags":        entry.Metadata.Tags,
		"license":     entry.Metadata.License,
		"owner":       entry.Metadata.Owner,
		"tools":       entry.Tools,
		"source":      entry.Source,
		"icon":        entry.Icon,
		"prompts":     entry.Prompts,
		"title":       entry.Title,
		"readme":      entry.Readme,
		"toolsUrl":    entry.ToolsURL,
		"dateAdded":   entry.DateAdded,
		"upstream":    entry.Upstream,
		"resources":   entry.Resources,
	}

	if entry.Metadata.Stars > 0 {
		publisherProvided["stars"] = entry.Metadata.Stars
	}

	server.Meta["io.modelcontextprotocol.registry/publisher-provided"] = publisherProvided
	// Note: io.modelcontextprotocol.registry/official metadata is added by the registry server
	// and should not be included in seed data

	if entry.Type == "remote" && entry.Remote != nil {
		transport := Transport{
			Type: entry.Remote.TransportType,
			URL:  entry.Remote.URL,
		}

		if entry.Remote.Headers != nil {
			for k, v := range entry.Remote.Headers {
				transport.Headers = append(transport.Headers, KeyValueInput{
					Name:  k,
					Value: v,
				})
			}
		}

		server.Remotes = []Transport{transport}
		return server
	}

	pkg := Package{
		RegistryType: "oci",
		Transport: Transport{
			Type: "stdio",
		},
	}

	if entry.Image != "" {
		// For OCI packages, the entire image reference (including tag/digest) goes in identifier
		// Don't set a separate version field for OCI packages
		pkg.Identifier = entry.Image
	}

	if len(entry.Env) > 0 {
		for _, env := range entry.Env {
			value := convertBraces(env.Value)
			kv := KeyValueInput{
				Name:  env.Name,
				Value: value,
			}

			if strings.Contains(env.Value, "{{") {
				variables := extractVars(env.Value, configMap)
				if len(variables) > 0 {
					kv.Variables = variables
					kv.IsRepeated = false
				}
			}

			pkg.EnvironmentVariables = append(pkg.EnvironmentVariables, kv)
		}
	}

	if len(entry.Secrets) > 0 {
		for _, secret := range entry.Secrets {
			varName := secret.Name
			kv := KeyValueInput{
				Name:  secret.Env,
				Value: fmt.Sprintf("{%s}", varName),
				Variables: map[string]InputSchema{
					varName: {
						IsSecret:   true,
						IsRequired: true,
					},
				},
			}
			pkg.EnvironmentVariables = append(pkg.EnvironmentVariables, kv)
		}
	}

	if len(entry.Command) > 0 {
		for _, cmd := range entry.Command {
			arg := Argument{}

			if strings.HasPrefix(cmd, "--") {
				parts := strings.SplitN(cmd, "=", 2)
				arg.Type = "named"
				arg.Name = parts[0]
				if len(parts) > 1 {
					arg.Value = convertBraces(parts[1])
					if strings.Contains(parts[1], "{{") {
						arg.Variables = extractVars(parts[1], configMap)
						arg.IsRepeated = false
					}
				}
			} else {
				arg.Type = "positional"
				arg.Value = cmd
			}

			pkg.PackageArguments = append(pkg.PackageArguments, arg)
		}
	}

	if len(entry.Volumes) > 0 {
		for _, vol := range entry.Volumes {
			value := convertBraces(vol)
			arg := Argument{
				Type:  "named",
				Name:  "-v",
				Value: value,
			}

			if strings.Contains(vol, "{{") {
				arg.Variables = extractVars(vol, configMap)
				arg.IsRepeated = false
			}

			pkg.RuntimeArguments = append(pkg.RuntimeArguments, arg)
		}
	}

	if entry.User != "" {
		value := convertBraces(entry.User)
		arg := Argument{
			Type:  "named",
			Name:  "-u",
			Value: value,
		}

		if strings.Contains(entry.User, "{{") {
			arg.Variables = extractVars(entry.User, configMap)
			arg.IsRepeated = false
		}

		pkg.RuntimeArguments = append(pkg.RuntimeArguments, arg)
	}

	// Set runtimeHint when runtimeArguments are present
	if len(pkg.RuntimeArguments) > 0 {
		pkg.RuntimeHint = "docker"
	}

	server.Packages = []Package{pkg}

	if entry.Upstream != "" {
		repoURL := entry.Upstream
		
		// Determine source identifier from URL (should be "github" or "gitlab", not a URL)
		sourceID := ""
		if strings.Contains(repoURL, "github.com") {
			sourceID = "github"
		} else if strings.Contains(repoURL, "gitlab.com") {
			sourceID = "gitlab"
		}
		
		server.Repository = &Repository{
			URL:    repoURL,
			Source: sourceID,
		}
	}
	return server
}

func convertBraces(value string) string {
	result := strings.ReplaceAll(value, "{{", "{")
	result = strings.ReplaceAll(result, "}}", "}")
	return result
}

func extractVars(value string, configMap map[string]map[string]interface{}) map[string]InputSchema {
	variables := make(map[string]InputSchema)

	start := 0
	for {
		idx := strings.Index(value[start:], "{{")
		if idx == -1 {
			break
		}
		idx += start

		endIdx := strings.Index(value[idx:], "}}")
		if endIdx == -1 {
			break
		}
		endIdx += idx

		varContent := value[idx+2 : endIdx]
		parts := strings.Split(varContent, "|")
		varName := parts[0]

		inputSchema := InputSchema{
			IsSecret: false,
			Format:   "string",
		}

		if configProp, ok := configMap[varName]; ok {
			if desc, ok := configProp["description"].(string); ok {
				inputSchema.Description = desc
			}
		}

		variables[varName] = inputSchema
		start = endIdx + 2
	}

	return variables
}
