package main

import (
	"fmt"
	"math/rand"
	"strings"
)

//MakeIndexMessage Generates a message to index this package
func MakeIndexMessage(pkg *Package) string {
	dependenciesNames := []string{}

	for _, dep := range pkg.Dependencies {
		dependenciesNames = append(dependenciesNames, dep.Name)
	}

	namesAsString := strings.Join(dependenciesNames, ",")
	return fmt.Sprintf("INDEX|%s|%s", pkg.Name, namesAsString)
}

//MakeRemoveMessage generates a message to remove a pakcage from the server's index
func MakeRemoveMessage(pkg *Package) string {
	return fmt.Sprintf("REMOVE|%s|", pkg.Name)
}

//MakeQueryMessage generates a message to check if a package is currently indexed
func MakeQueryMessage(pkg *Package) string {
	return fmt.Sprintf("QUERY|%s|", pkg.Name)
}

var possibleInvalidCommands = []string{"BLINDEX", "REMOVES", "QUER", "LIZARD", "I"}
var possibleInvalidChars = []string{"=", "+", "☃", " "}

//MakeBrokenMessage returns a message that's somehow broken and should be rejected
//by the server
func MakeBrokenMessage() string {
	syntaxError := rand.Intn(10)%2 == 0

	if syntaxError {
		invalidChar := possibleInvalidChars[rand.Intn(len(possibleInvalidChars))]
		return fmt.Sprintf("INDEX|emacs%selisp", invalidChar)
	}

	invalidCommand := possibleInvalidCommands[rand.Intn(len(possibleInvalidCommands))]
	return fmt.Sprintf("%s|a|b", invalidCommand)
}
