package main

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	//LineFormat defines a valid input line
	// see data.go and data/brew-dependencies.txt
	LineFormat = "^\\S+:( +)?(\\S+ *)*"
)

var (
	//Matches well-formed lines from the data file
	lineMatcher, _ = regexp.Compile(LineFormat)
)

//Package represents a package and its dependencies
type Package struct {
	Name         string
	Dependencies []*Package
}

//AddDependency makes this package depend on some other
func (pkg *Package) AddDependency(to *Package) {
	pkg.Dependencies = append(pkg.Dependencies, to)
}

//AllPackages is a repository for all known packages
type AllPackages struct {
	//All packages we know of
	Packages []*Package
}

// Names returns the names of all known packages
func (allPackages *AllPackages) Names() []string {
	names := []string{}
	for _, p := range allPackages.Packages {
		names = append(names, p.Name)
	}
	return names
}

// Named finds or creates a package with given name. This should be the only
// function used to instantiate packages in production so that we can
// keep a single instance per package.
func (allPackages *AllPackages) Named(name string) *Package {
	var pkg *Package

	for _, p := range allPackages.Packages {
		if p.Name == name {
			pkg = p
		}
	}

	if pkg == nil {
		pkg = makeUnprocessedPackage(name)
		allPackages.Packages = append(allPackages.Packages, pkg)
	}

	return pkg
}

func makeUnprocessedPackage(name string) *Package {
	return &Package{
		Name:         name,
		Dependencies: make([]*Package, 0),
	}
}

// TokeniseLine parses a single line from the text
// file, in the format of LineFormat.
// It returns the relevant tokens as an array. The
// first element of the array is the package name,
// any subsequent elements are dependencies.
func TokeniseLine(line string) ([]string, error) {
	if !lineMatcher.MatchString(line) {
		return nil, fmt.Errorf("Invalid line: %#v", line)
	}

	sanitisedLine := strings.Replace(strings.Trim(line, " "), "  ", " ", -1)
	tokens := strings.Split(sanitisedLine, " ")

	packageName := strings.TrimRight(tokens[0], ":")

	dependenciesNames := tokens[1:len(tokens)]
	return append([]string{packageName}, dependenciesNames...), nil
}

// TokensToPackage converts an array of tokens to a Package.
// The first element of the token must be the package name,
// any following elmeents will be dependencies.
func TokensToPackage(allPackages *AllPackages, tokens []string) (*Package, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("Passed in empty array of tokens")
	}

	pkg := allPackages.Named(tokens[0])
	for _, dep := range tokens[1:len(tokens)] {
		depPackage := allPackages.Named(dep)
		pkg.AddDependency(depPackage)
	}

	return pkg, nil
}

// TextToPackages parses a string containing a sequence of lines as per the
// TokeniseLine function and adds all parsed contents to a AllPackages instance.
func TextToPackages(allPackages *AllPackages, text string) (*AllPackages, error) {
	lines := strings.Split(text, "\n")

	for _, l := range lines {
		if len(l) == 0 {
			continue
		}

		tokens, err := TokeniseLine(l)
		if err != nil {
			return nil, err
		}

		_, err = TokensToPackage(allPackages, tokens)
		if err != nil {
			return nil, err
		}
	}

	return allPackages, nil
}

// BrewToPackages converts a homebrew index
// dump into packages.
func BrewToPackages(allPackages *AllPackages) (*AllPackages, error) {
	dataFile := "data/brew-dependencies.txt"
	data, err := Asset(dataFile)
	if err != nil {
		return nil, fmt.Errorf("Data file [%s] not embedded!", dataFile)
	}

	dataAsString := string(data[:])

	pkgs, err := TextToPackages(allPackages, dataAsString)

	if err != nil {
		return nil, err
	}

	return pkgs, nil
}

//SegmentListPackages breaks a list of packages in N segments,
// where N <= maxNumberOfSegments
func SegmentListPackages(fullList []*Package, maxNumberOfSegments int) [][]*Package {
	fullListSize := len(fullList)
	result := [][]*Package{}

	if maxNumberOfSegments < 1 {
		return append(result, fullList)
	}

	if maxNumberOfSegments > fullListSize {
		maxNumberOfSegments = fullListSize
	}

	optimalNumberOfElementsPerSegment := fullListSize / maxNumberOfSegments

	beginning := 0
	for i := 0; i < (maxNumberOfSegments - 1); i++ {
		end := beginning + optimalNumberOfElementsPerSegment
		slice := fullList[beginning:end]
		result = append(result, slice)
		beginning = beginning + optimalNumberOfElementsPerSegment
	}
	if beginning < fullListSize {
		result = append(result, fullList[beginning:])
	}

	return result
}
