package model

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
)

type Version struct {
	Prefix       string
	Major        int
	Minor        int
	Maintenance  int
	Suffix       string
	BuildDate    string
	MinGoVersion string
}

// The compatibility list.
var frameworkCompatibleRangeList = [][]string{
	{"0.0.0", "0.20.0"},   // minimum Revel version to use with this version of the tool
	{"0.19.99", "0.30.0"}, // Compatible with Framework V 0.19.99 - 0.30.0
	{"1.0.0", "1.9.0"},    // Compatible with Framework V 1.0 - 1.9
}

// Parses a version like v1.2.3a or 1.2.
var versionRegExp = regexp.MustCompile(`([^\d]*)?([0-9]*)\.([0-9]*)(\.([0-9]*))?(.*)`)

// Parse the version and return it as a Version object.
func ParseVersion(version string) (v *Version, err error) {
	v = &Version{}
	return v, v.ParseVersion(version)
}

// Parse the version and return it as a Version object.
func (v *Version) ParseVersion(version string) (err error) {
	parsedResult := versionRegExp.FindAllStringSubmatch(version, -1)
	if len(parsedResult) != 1 {
		err = errors.Errorf("Invalid version %s", version)
		return
	}
	if len(parsedResult[0]) != 7 {
		err = errors.Errorf("Invalid version %s", version)
		return
	}

	v.Prefix = parsedResult[0][1]
	v.Major = v.intOrZero(parsedResult[0][2])
	v.Minor = v.intOrZero(parsedResult[0][3])
	v.Maintenance = v.intOrZero(parsedResult[0][5])
	v.Suffix = parsedResult[0][6]

	return
}

// Returns 0 or an int value for the string, errors are returned as 0.
func (v *Version) intOrZero(input string) (value int) {
	if input != "" {
		value, _ = strconv.Atoi(input)
	}
	return value
}

// Returns true if this major revision is compatible.
func (v *Version) CompatibleFramework(c *CommandConfig) error {
	for i, rv := range frameworkCompatibleRangeList {
		start, _ := ParseVersion(rv[0])
		end, _ := ParseVersion(rv[1])
		if !v.Newer(start) || v.Newer(end) {
			continue
		}

		// Framework is older then 0.20, turn on historic mode
		if i == 0 {
			c.HistoricMode = true
		}
		return nil
	}
	return errors.New("Tool out of date - do a 'go get -u github.com/revel/cmd/revel'")
}

// Returns true if this major revision is newer then the passed in.
func (v *Version) MajorNewer(o *Version) bool {
	if v.Major != o.Major {
		return v.Major > o.Major
	}
	return false
}

// Returns true if this major or major and minor revision is newer then the value passed in.
func (v *Version) MinorNewer(o *Version) bool {
	if v.Major != o.Major {
		return v.Major > o.Major
	}
	if v.Minor != o.Minor {
		return v.Minor > o.Minor
	}
	return false
}

// Returns true if the version is newer then the current on.
func (v *Version) Newer(o *Version) bool {
	if v.Major != o.Major {
		return v.Major > o.Major
	}
	if v.Minor != o.Minor {
		return v.Minor > o.Minor
	}
	if v.Maintenance != o.Maintenance {
		return v.Maintenance > o.Maintenance
	}
	return true
}

// Convert the version to a string.
func (v *Version) VersionString() string {
	return fmt.Sprintf("%s%d.%d.%d%s", v.Prefix, v.Major, v.Minor, v.Maintenance, v.Suffix)
}

// Convert the version build date and go version to a string.
func (v *Version) String() string {
	return fmt.Sprintf("Version: %s%d.%d.%d%s\nBuild Date: %s\n Minimum Go Version: %s",
		v.Prefix, v.Major, v.Minor, v.Maintenance, v.Suffix, v.BuildDate, v.MinGoVersion)
}
