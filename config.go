package main

import (
	"fmt"
	"github.com/go-yaml/yaml"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

type Config struct {
	OpenAI struct {
		APIKey string `yaml:"api_key"`
	} `yaml:"openai"`
}

var homeDir, _ = os.UserHomeDir()
var configFilePath = filepath.Join(homeDir, "ai.yaml")

func getAPIKey() string {
	apiKey := readAPIKey()
	if apiKey == "" {
		apiKey = initApiKey()
	}
	return apiKey
}

func readAPIKey() string {
	// Check if the API key is set in the environment variable
	envAPIKey := os.Getenv("OPENAI_API_KEY")
	if envAPIKey != "" {
		return envAPIKey
	}

	configFile, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return ""
	}

	var config Config
	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		log.Fatalf("Error unmarshalling config file: %v", err)
	}

	return config.OpenAI.APIKey
}

func askAPIKey() string {
	var apiKey string
	fmt.Print("Enter your OpenAI API Key (configuration will be updated): ")
	fmt.Scanln(&apiKey)
	return apiKey
}

func writeAPIKey(apiKey string) {
	config := Config{
		OpenAI: struct {
			APIKey string `yaml:"api_key"`
		}{
			APIKey: apiKey,
		},
	}

	configData, err := yaml.Marshal(config)
	if err != nil {
		log.Fatalf("Error marshalling config data: %v", err)
	}

	err = ioutil.WriteFile(configFilePath, configData, 0644)
	if err != nil {
		log.Fatalf("Error writing config file: %v", err)
	}

	fmt.Printf("API key added to your %s\n", configFilePath)
}

func initApiKey() string {
	fmt.Printf("Please provide your OpenAI API.\n"+
		"- Through an environment variable: OPENAI_API_KEY\n"+
		"- Through a configuration file:    %s\n", configFilePath)
	var apiKey = askAPIKey()
	writeAPIKey(apiKey)
	return apiKey
}
