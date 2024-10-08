package main

// trying to build api fuzzer: part 1
// three sections:
// poet: generate the api fuzzing code
// courier: send the requests
// oracle: check the responses

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	content, err := ioutil.ReadFile("panicpanda.txt")
	if err != nil {
		fmt.Println("Error reading swagger file: ", err)
		return
	}
	fmt.Println(string(content))
	fmt.Println("Input domain and paths (everything before the paths on the swaggerdoc):")
	controllerAddress, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading controller address: ", err)
		return
	}
	controllerAddress = strings.TrimSpace(controllerAddress)
	fmt.Println("Input auth token (if none, leave blank):")
	token, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading token: ", err)
		return
	}
	authflag := true
	if token == "" {
		fmt.Println("No token provided. Will not use token.")
		authflag = false
	}
	token = strings.TrimSpace(token)
	fmt.Println("Input timer:")
	timea, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading timer: ", err)
		return
	}
	timea = strings.TrimSpace(timea)
	if timea == "" {
		fmt.Println("No timer provided. Will use default timer of 10 seconds.")
		timea = "10"
	}
	timer, err := strconv.Atoi(timea)
	if err != nil {
		fmt.Println("Error converting timer:", err)
		return
	}
	var wordlist []string
	var pathlist []string
	fmt.Println("Input wordlist file path: (if you want pure random data, leave blank)")
	wordlistPath, err := reader.ReadString('\n')
	wordlistPath = strings.TrimSpace(wordlistPath)
	if err != nil {
		fmt.Println("Error reading wordlist file:", err)
		return
	}
	if wordlistPath != "" {
		wordListFile, err := os.Open(wordlistPath)
		if err != nil {
			fmt.Println("Error opening wordlist file:", err)
			return
		}
		scanner := bufio.NewScanner(wordListFile)
		pathflag := false
		for scanner.Scan() {
			word := strings.TrimSpace(scanner.Text())
			if word == "PATHLIST" {
				pathflag = true
				continue
			}
			if word != "" {
				if pathflag {
					pathlist = append(pathlist, word)
				} else {
					wordlist = append(wordlist, word)
				}
			}
		}

		defer wordListFile.Close()
	}
	fmt.Println("Is your swaggerdoc v2 or v3?")
	version := ""
	for {
		version, err = reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading version: ", err)
		}
		version = strings.TrimSpace(version)
		if version == "v2" || version == "v3" {
			break
		} else {
			fmt.Println("Please enter either v2 or v3")
		}
	}
	fmt.Println("Input swagger file path:")
	swagstr, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading swagger file: ", err)
		return
	}
	swagstr = strings.TrimSpace(swagstr)
	var swag []apiDoc
	if version == "v2" {
		swag = swag2(swagstr)
	} else {
		swag = swag3(swagstr)

	}
	if swag == nil {
		fmt.Println("Error parsing swagger file")
		return
	}
	headers := false
	fmt.Println("Do you want to fuzz the headers? (Y/N)")
	log, err := reader.ReadString('\n')
	if err != nil && !(log == "y" || log == "n" || log == "Y" || log == "N") {
		fmt.Println("Error reading log option:", err)
		return
	} else if log == "y" || log == "Y" {
		headers = true
	}

	backoff := 0
	fmt.Println("How many seconds do you want to wait before retrying the fuzzer after continuous failure?")
	backoffstr, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading backoff time: ", err)
		return
	}
	backoffstr = strings.TrimSpace(backoffstr)
	if backoffstr == "" {
		fmt.Println("No backoff time provided. Will use default backoff time of 10 seconds.")
		backoffstr = "10"
	}
	backoff, err = strconv.Atoi(backoffstr)
	if err != nil {
		fmt.Println("Error converting backoff time: ", err)
		return
	}
	threadManager(controllerAddress, swag, token, timer, authflag, headers, wordlist, pathlist, backoff)
}

func threadManager(controllerAddress string, apiList []apiDoc, args string, timer int, requiresAuth bool, headers bool, wordlist []string, pathlist []string, backoff int) {
	var wrkgrp sync.WaitGroup
	timeout, cancel := context.WithTimeout(context.Background(), time.Duration(timer)*time.Second)
	defer cancel()
	fmt.Printf("Starting the fuzzer for %d seconds\n", timer)
	id := 0
	var printMutex sync.Mutex
	defang := true //DEFANG FLAG HERE
	for _, api := range apiList {
		wrkgrp.Add(1)
		go func(id int) {
			defer wrkgrp.Done()
			if !defang {
				fullfunc(controllerAddress, api, args, timer, requiresAuth, headers, id, timeout, &printMutex, wordlist, pathlist, backoff)
			} else {
				demo(controllerAddress, api, args, timer, requiresAuth, headers, id, timeout, &printMutex, wordlist, pathlist, backoff)
			}
		}(id)
		id++
	}
	wrkgrp.Wait()
}
