package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/libcfbuildpack/helper"

	"github.com/Masterminds/semver"
)

var LTS = map[string]int{
	"argon":   4,
	"boron":   6,
	"carbon":  8,
	"dubnium": 10,
}

func LoadNvmrc(context detect.Detect) (string, error) {
	if nvmrcExists, err := helper.FileExists(filepath.Join(context.Application.Root, ".nvmrc")); err != nil {
		return "", err
	} else if !nvmrcExists {
		return "", nil
	}

	nvmrcContents, err := ioutil.ReadFile(filepath.Join(context.Application.Root, ".nvmrc"))
	if err != nil {
		return "", err
	}

	nvmrcVersion, err := validateNvmrc(string(nvmrcContents))
	if err != nil {
		return "", err
	}

	if nvmrcVersion == "node" {
		context.Logger.Info(".nvmrc specified latest node version, this will be selected from versions available in buildpack.toml")
	}

	if strings.HasPrefix(nvmrcVersion, "lts") {
		context.Logger.Info(".nvmrc specified an lts version, this will be selected from versions available in buildpack.toml")
	}

	return formatNvmrcContent(nvmrcVersion), nil
}

func WarnNodeEngine(nvmrcNodeVersion string, packageJSONNodeVersion string) []string {
	docsLink := "http://docs.cloudfoundry.org/buildpacks/node/node-tips.html"

	var logs []string
	if nvmrcNodeVersion != "" && packageJSONNodeVersion == "" {
		logs = append(logs, fmt.Sprintf("Using the node version specified in your .nvmrc See: %s", docsLink))
	}
	if packageJSONNodeVersion != "" && nvmrcNodeVersion != "" {
		logs = append(logs, fmt.Sprintf("Node version in .nvmrc ignored in favor of 'engines' field in package.json"))
	}
	if packageJSONNodeVersion == "" && nvmrcNodeVersion == "" {
		logs = append(logs, fmt.Sprintf("Node version not specified in package.json or .nvmrc. See: %s", docsLink))
	}
	if packageJSONNodeVersion == "*" {
		logs = append(logs, fmt.Sprintf("Dangerous semver range (*) in engines.node. See: %s", docsLink))
	}
	if strings.HasPrefix(packageJSONNodeVersion, ">") {
		logs = append(logs, fmt.Sprintf("Dangerous semver range (>) in engines.node. See: %s", docsLink))
	}
	return logs
}

func formatNvmrcContent(version string) string {
	if version == "node" {
		return "*"
	} else if strings.HasPrefix(version, "lts") {
		ltsName := strings.Split(version, "/")[1]
		if ltsName == "*" {
			maxVersion := 0
			for _, versionValue := range LTS {
				if maxVersion < versionValue {
					maxVersion = versionValue
				}
			}
			return strconv.Itoa(maxVersion) + ".*.*"
		} else {
			versionNumber := LTS[ltsName]
			return strconv.Itoa(versionNumber) + ".*.*"
		}
	} else {
		matcher := regexp.MustCompile(semver.SemVerRegex)

		groups := matcher.FindStringSubmatch(version)
		for index := 0; index < len(groups); index++ {
			if groups[index] == "" {
				groups = append(groups[:index], groups[index+1:]...)
				index--
			}
		}

		return version + strings.Repeat(".*", 4-len(groups))
	}
}

func validateNvmrc(content string) (string, error) {
	content = strings.TrimSpace(strings.ToLower(content))

	if content == "lts/*" || content == "node" {
		return content, nil
	}

	for key, _ := range LTS {
		if content == strings.ToLower("lts/"+key) {
			return content, nil
		}
	}

	if len(content) > 0 && content[0] == 'v' {
		content = content[1:]
	}

	if _, err := semver.NewVersion(content); err != nil {
		return "", fmt.Errorf("invalid version %s specified in .nvmrc", err)
	}

	return content, nil
}
