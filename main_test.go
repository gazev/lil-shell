package main

import (
	"fmt"
	"log"
	"reflect"
	"testing"
)

// NOTES: apparently, when a function is returning an empty slice it is just nil

var tests1 = map[string][]string{
	"\n":                                  nil,
	" echo hi \n":                         {"echo", "hi"},
	" echo hi \r\n":                       {"echo", "hi"},
	" ps -ef | \t  less \r\n":             {"ps", "-ef", "|", "less"},
	"cat main.go | awk '{print $1 $2}'\n": {"cat", "main.go", "|", "awk", "{print $1 $2}"},
	`echo "hello world" | sed --expression='s/hello/hi/g'`: {"echo", `"hello world"`, "|", "sed", "--expression=s/hello/hi/g"},
	"ls -lsa | column -t | awk '{print $10}'":              {"ls", "-lsa", "|", "column", "-t", "|", "awk", "{print $10}"},
}

func TestParseInput(t *testing.T) {
	fmt.Println("Testing ParseInput")
	i := 1
	for k, v := range tests1 {
		res := ParseInput(k)
		if !reflect.DeepEqual(res, v) {
			fmt.Printf("\033[0;31mTest %d FAIL\033[0m\n", i)
			fmt.Printf("Test: %s\n", k)
			fmt.Printf("Expected: %s\n", v)
			fmt.Printf("Got: %s\n", res)
			log.Fatalf("Failed %d\n", i)
		} else {
			fmt.Printf("\033[0;32mTests %d OK\033[0m\n", i)
		}
		i++
	}
}

var tests2 = map[string][][]string{
	"\n":              nil,
	"echo hi":         {{"echo", "hi"}},
	"ps -ef | less\n": {{"ps", "-ef"}, {"less"}},
	"ps aux  |   grep go  | awk '{print $1 $2}'": {{"ps", "aux"}, {"grep", "go"}, {"awk", "{print $1 $2}"}},
}

func TestGetPipeSeparatedCommands(t *testing.T) {
	fmt.Println("Testing GetPipeSeparatedCommands")
	i := 1
	for k, v := range tests2 {
		parsedInput := ParseInput(k)
		res := GetPipeSeparatedCommands(parsedInput)
		if !reflect.DeepEqual(res, v) {
			fmt.Printf("\033[0;31mTest %d FAIL\033[0m\n", i)
			fmt.Printf("Test: %s\n", k)
			fmt.Printf("Expected: %s\n", v)
			fmt.Printf("Got: %s\n", res)
			log.Fatalf("Failed %d\n", i)
		} else {
			fmt.Printf("\033[0;32mTests %d OK\033[0m\n", i)
		}
		i++
	}
}
