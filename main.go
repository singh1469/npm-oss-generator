package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
)

var (
	buf    bytes.Buffer
	logger = log.New(&buf, "logger: ", log.Lshortfile)
)

// Dependency type
type Dependency struct {
	Name        string
	Version     string
	Description string
	Homepage    string
	AbsPath     string
	License     string
}

// Direct Dependency type
type DirectDependency struct {
	Dependencies    map[string]interface{}
	DevDependencies map[string]interface{}
}

func getJson(path string, whiteList []string, c chan Dependency) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		logger.Panic("Read file error", path, err)
	}
	d := Dependency{}

	jsonErr := json.Unmarshal(b, &d)
	if jsonErr != nil {
		// license field/key is probably is not a string
		m := make(map[string]interface{})
		err = json.Unmarshal(b, &m)
		if jsonErr == nil {
			logger.Panic("\n Error parsing package.json:", jsonErr, path)
		}
		// assume string values
		d.Name = m["name"].(string)
		d.Version = m["version"].(string)
		d.Description = m["description"].(string)
		d.Homepage = m["homepage"].(string)

		// licenses can be of a varied type
		rt := reflect.TypeOf(m["license"])
		switch rt.Kind() {
		case reflect.String:
			d.License = m["license"].(string)
		case reflect.Map:
			mapLicense := m["license"].(map[string]interface{})
			d.License = mapLicense["type"].(string)
		case reflect.Slice:
			arrayLicense := m["license"].([]interface{})
			d.License = arrayLicense[0].(string)
		default:
			logger.Panic("License property type is neither a string, map or array. Found type:", rt.Kind())
		}
	}

	// are we interested in specific packages only
	if len(whiteList) > 0 {
		// verify whether pkg is in white list
		found := false
		for _, v := range whiteList {
			if v == d.Name {
				found = true
				break
			}
		}
		if found != true {
			return
		}
	}

	d.AbsPath = path
	//logger.Print(d, "\n")
	//logger.Printf("%+v\n", d)
	c <- d
}

func pathWalk(root string) ([]string, error) {

	var deps []string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}

		// ignore root
		if info.IsDir() && path == root {
			return nil
		}

		// ignore directories
		if info.IsDir() {
			return nil
		}

		if info.Name() == "package.json" {
			deps = append(deps, path)
		}

		return nil
	})

	if len(deps) < 1 {
		return nil, errors.New("Failed to find any dependencies! Check your node_modules path.")
	}
	return deps, nil
}

func getDependencies(path string) (DirectDependency, error) {
	var str bytes.Buffer
	str.WriteString(path)
	str.WriteString("/../package.json")
	fullPath := filepath.FromSlash(str.String())
	srcJson, err := ioutil.ReadFile(fullPath)
	if err != nil {
		logger.Printf("prevent panic by handling failure accessing a root package.json %q: %v\n", path, err)
		return DirectDependency{}, err
	}

	d := DirectDependency{}
	err = json.Unmarshal(srcJson, &d)
	if err != nil {
		logger.Printf("prevent panic by handling failure when parsing root package.json to map %q: %v\n", path, err)
		return DirectDependency{}, err
	}
	return d, nil
}

func isValidPath(path string) (bool, error) {
	if path == "" {
		logger.Panic("Directory path is invalid")
	}
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, errors.New("Failed to find path.")
		}
		return false, err
	}

	return true, nil
}

func main() {
	log.SetOutput(os.Stdout)
	directDepsOnly := flag.Bool("onlyDirectDependencies", false, "Get direct dependencies only")
	directDevDepsOnly := flag.Bool("onlyDirectDevDependencies", false, "Get direct dev dependencies only")
	output := flag.String("output", "", "Save to a file")
	pathToDeps := flag.String("path", "", "Path to node_modules directory")
	flag.Parse()

	// validate path
	_, err := isValidPath(*pathToDeps)
	if err != nil {
		logger.Panic(err)
	}

	var dependencyWhitelist []string
	if *directDepsOnly == true {
		d, err := getDependencies(*pathToDeps)
		if err != nil {
			logger.Panic(err)
		}
		keys := reflect.ValueOf(d.Dependencies).MapKeys()
		for _, v := range keys {
			dependencyWhitelist = append(dependencyWhitelist, v.Interface().(string))
		}
	}

	if *directDevDepsOnly == true {
		d, err := getDependencies(*pathToDeps)
		if err != nil {
			logger.Panic(err)
		}
		keys := reflect.ValueOf(d.DevDependencies).MapKeys()
		for _, v := range keys {
			dependencyWhitelist = append(dependencyWhitelist, v.Interface().(string))
		}
	}

	deps, err := pathWalk(*pathToDeps)
	if err != nil {
		panic(err)
	}
	ch := make(chan Dependency)
	go func() {
		for _, v := range deps {
			go getJson(string(v), dependencyWhitelist, ch)
		}
	}()

	depsCount := len(deps)
	if len(dependencyWhitelist) > 0 {
		depsCount = len(dependencyWhitelist)
	}
	processed := 0
	var s []Dependency
chListener:
	for {
		select {
		case msg := <-ch:
			//fmt.Printf("%+v\n", msg)
			//msgJson, _ := json.Marshal(msg)
			//fmt.Print(reflect.TypeOf(msgJson))
			s = append(s, msg)
			processed++
			if processed == depsCount {
				break chListener
			}
		}
	}

	// sort by dependency name
	sort.Slice(s, func(i, j int) bool { return s[i].Name < s[j].Name })
	var b bytes.Buffer
	b.WriteString("[")
	for i, v := range s {
		msgJson, _ := json.Marshal(v)
		json.Indent(&b, msgJson, "", "\t")
		if i < len(s)-1 {
			b.WriteString(",")
		}
	}
	b.WriteString("]")
	switch *output {
	case "file":
		f, err := os.Create("./out.json")
		if err != nil {
			logger.Panic(err)
		}
		f.Write(b.Bytes())
		f.Close()
	default:
		b.WriteTo(os.Stdout)
	}
	logger.Print("\nNothing more to to.")
	fmt.Print(&buf)
	return
}
