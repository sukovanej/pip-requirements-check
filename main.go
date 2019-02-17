package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

func main() {
	file := flag.String("file", "./requirements.txt", "requirements file path")
	majorOnly := flag.Bool("major", false, "print major changes only")
	flag.Parse()

	contentBytes, err := ioutil.ReadFile(*file)
	if err != nil {
		panic(err)
	}

	content := string(contentBytes[:])
	packages := GetPackages(content)

	result := make(chan Result, 0)

	for _, pkg := range packages {
		go GetPackageLastVersion(pkg, result)
	}

	pkgLen := len(packages)
	count := 0
	for r := range result {
		diff := VersionDiffOrder(r.Pkg.Version, r.Version)

		if diff < 3 && diff != -1 && (!*majorOnly || diff == 0) {
			fmt.Print("(", (count * 100 / pkgLen), "%) ")
			if diff == 0 {
				c := color.New(color.FgRed)
				c.Print("(Major)")
			} else if diff > 0 {
				c := color.New(color.FgYellow)
				c.Print("(Minor)")
			}
			fmt.Println(" ", r.Pkg.Name, ": ", r.Pkg.Version, " -> ", r.Version)
		}

		count++
		if count == pkgLen {
			close(result)
		}
	}
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

func GetPackageLastVersion(pkg PackageVersion, result chan Result) {
	resp, err := http.Get("https://pypi.org/pypi/" + pkg.Name + "/json")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
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

	result <- Result{pkg, str}
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
