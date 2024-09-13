package main

type Message struct {
	Role    string `yaml:"role" json:"role"`
	Content string `yaml:"content" json:"content"`
}

type ReturnCommandFunction struct {
	Command  string   `json:"command"`
	Binaries []string `json:"binaries"`
}
