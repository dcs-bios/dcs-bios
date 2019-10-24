// semver-to-msiversion.go takes a version like "0.1.2-alpha3" (first command line argument)
// and a build number (second command line argument), ignores the prerelease version part ("-alpha3")
// and prints "major.minor.patch.buildNumber"
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	semver "github.com/Masterminds/semver"
)

func main() {
	sv := semver.MustParse(os.Args[1])
	buildNumber, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatal("invalid build number:", os.Args[2])
	}
	fmt.Printf("%d.%d.%d.%d", sv.Major(), sv.Minor(), sv.Patch(), buildNumber)
}
