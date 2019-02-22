package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/fatih/color"
)

func main() {
	file := flag.String("file", "./requirements.txt", "requirements file path")
	majorOnly := flag.Bool("major", false, "print major changes only")
	pypiUrl := flag.String("pypi", "https://pypi.org/pypi/", "pypi url")
	flag.Parse()

	contentBytes, err := ioutil.ReadFile(*file)
	if err != nil {
		panic(err)
	}

	content := string(contentBytes[:])
	packages := GetPackages(content)

	wg := &sync.WaitGroup{}

	for _, pkg := range packages {
		wg.Add(1)
		go GetPackageLastVersion(*pypiUrl, pkg, wg, func(r Result) {
			diff := VersionDiffOrder(r.Pkg.Version, r.Version)
			if diff < 3 && diff != -1 && (!*majorOnly || diff == 0) {
				if diff == 0 {
					c := color.New(color.FgRed)
					c.Print("(Major)")
				} else if diff > 0 {
					c := color.New(color.FgYellow)
					c.Print("(Minor)")
				}
				fmt.Println(" ", r.Pkg.Name, ": ", r.Pkg.Version, " -> ", r.Version)
			}
		})
	}

	wg.Wait()
}

type PackageVersion struct {
	Name    string
	Version string
}

type Result struct {
	Pkg     PackageVersion
	Version string
}

func GetPackages(input string) []PackageVersion {
	result := make([]PackageVersion, 0)
	for _, line := range strings.Split(input, "\n") {
		if line == "" || line[0] == '#' || line[0] == '-' {
			continue
		}

		pkgVer := strings.Split(strings.Split(line, " ")[0], "==")
		result = append(result, PackageVersion{pkgVer[0], pkgVer[1]})
	}

	return result
}

type PypiResponse struct {
	Releases map[string]struct{} `json:"releases"`
}

func GetPackageLastVersion(pypiUrl string, pkg PackageVersion, wg *sync.WaitGroup, callback func(Result)) {
	resp, err := http.Get(pypiUrl + pkg.Name + "/json")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if resp.StatusCode == 404 {
		wg.Done()
		callback(Result{pkg, "not found"})
		return
	}
	if err != nil {
		panic(err)
	}
	var pypiResp PypiResponse
	json.Unmarshal(body, &pypiResp)
	str := "0.0"
	for k := range pypiResp.Releases {
		if IsGreaterVersion(k, str) {
			str = k
		}
	}

	callback(Result{pkg, str})
	wg.Done()
}

func IsGreaterVersion(v1, v2 string) bool {
	v1List, v2List := strings.Split(v1, "."), strings.Split(v2, ".")
	for i := 0; i < min(len(v1List), len(v2List)); i++ {
		n, _ := strconv.ParseInt(v1List[i], 10, 64)
		m, _ := strconv.ParseInt(v2List[i], 10, 64)

		if n > m {
			return true
		} else {
			return false
		}
	}
	return true
}

func VersionDiffOrder(v1, v2 string) int {
	v1List, v2List := strings.Split(v1, "."), strings.Split(v2, ".")
	for i := 0; i < min(len(v1List), len(v2List)); i++ {
		n, _ := strconv.ParseInt(v1List[i], 10, 64)
		m, _ := strconv.ParseInt(v2List[i], 10, 64)

		if n != m {
			return i
		}
	}
	return -1
}

func min(a, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}
