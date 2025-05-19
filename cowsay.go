package cowsay

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Say to return cowsay string.
func Say(phrase string, options ...Option) (string, error) {
	cow, err := New(options...)
	if err != nil {
		return "", err
	}
	return cow.Say(phrase)
}

// LocationType indicates the type of COWPATH.
type LocationType int

const (
	// InBinary indicates the COWPATH in binary.
	InBinary LocationType = iota

	// InDirectory indicates the COWPATH in your directory.
	InDirectory
)

// CowPath is information of the COWPATH.
type CowPath struct {
	// Name is name of the COWPATH.
	// If you specified `COWPATH=/foo/bar`, Name is `/foo/bar`.
	Name string
	// CowFiles are name of the cowfile which are trimmed ".cow" suffix.
	CowFiles []string
	// LocationType is the type of COWPATH
	LocationType LocationType
}

// Lookup will look for the target cowfile in the specified path.
// If it exists, it returns the cowfile information and true value.
func (c *CowPath) Lookup(target string) (*CowFile, bool) {
	for _, cowfile := range c.CowFiles {
		if cowfile == target {
			return &CowFile{
				Name:         cowfile,
				BasePath:     c.Name,
				LocationType: c.LocationType,
			}, true
		}
	}
	return nil, false
}

// CowFile is information of the cowfile.
type CowFile struct {
	// Name is name of the cowfile.
	Name string
	// BasePath is the path which the cowpath is in.
	BasePath string
	// LocationType is the type of COWPATH
	LocationType LocationType
}

// ReadAll reads the cowfile content.
// If LocationType is InBinary, the file read from binary.
// otherwise reads from file system.
func (c *CowFile) ReadAll() ([]byte, error) {
	if c.LocationType == InBinary {
		// go embed is used "/" separator
		joinedPath := path.Join(c.BasePath, c.Name+".cow")
		return Asset(joinedPath)
	}
	joinedPath := filepath.Join(c.BasePath, c.Name+".cow")
	return os.ReadFile(joinedPath)
}

// Cows to get list of cows
func Cows() ([]*CowPath, error) {
	cowPaths, err := cowsFromCowPath()
	if err != nil {
		return nil, err
	}
	cowPaths = append(cowPaths, &CowPath{
		Name:         "cows",
		CowFiles:     CowsInBinary(),
		LocationType: InBinary,
	})
	return cowPaths, nil
}

func cowsFromCowPath() ([]*CowPath, error) {
	cowPaths := make([]*CowPath, 0)
	cowPath := os.Getenv("COWPATH")
	if cowPath == "" {
		return cowPaths, nil
	}
	paths := splitPath(cowPath)
	for _, path := range paths {
		dirEntries, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}
		path := &CowPath{
			Name:         path,
			CowFiles:     []string{},
			LocationType: InDirectory,
		}
		for _, entry := range dirEntries {
			name := entry.Name()
			if strings.HasSuffix(name, ".cow") {
				name = strings.TrimSuffix(name, ".cow")
				path.CowFiles = append(path.CowFiles, name)
			}
		}
		sort.Strings(path.CowFiles)
		cowPaths = append(cowPaths, path)
	}
	return cowPaths, nil
}

// GetCow to get cow's ascii art
func (cow *Cow) GetCow() (string, error) {
	src, err := cow.typ.ReadAll()
	if err != nil {
		return "", err
	}

	// Parse color variables from the cow file
	colorVars := make(map[string]string)
	lines := strings.Split(string(src), "\n")

	// First pass: collect all variable definitions
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "$") && strings.Contains(line, "=") && !strings.Contains(line, "$the_cow") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				varName := strings.TrimSpace(parts[0])
				varValue := strings.TrimSpace(parts[1])

				// Remove comments but keep the rest of the value
				if idx := strings.Index(varValue, "#"); idx != -1 {
					varValue = varValue[:idx]
					varValue = strings.TrimSpace(varValue)
				}

				// Extract the quoted part including spaces
				if strings.HasPrefix(varValue, "\"") && strings.HasSuffix(varValue, "\";") {
					varValue = strings.TrimSuffix(strings.TrimPrefix(varValue, "\""), "\";")
				} else if strings.HasPrefix(varValue, "\"") && strings.HasSuffix(varValue, "\"") {
					varValue = strings.TrimSuffix(strings.TrimPrefix(varValue, "\""), "\"")
				}

				// Replace \e with ESC character
				varValue = strings.ReplaceAll(varValue, "\\e", "\033")

				colorVars[varName] = varValue
			}
		}
	}

	// Second pass: resolve nested variable references
	// We might need multiple passes to resolve variables that reference other variables
	maxPasses := 5 // Prevent infinite loops
	for i := 0; i < maxPasses; i++ {
		madeChanges := false
		for varName, varValue := range colorVars {
			// Replace $thoughts in variable values
			if strings.Contains(varValue, "$thoughts") {
				newValue := strings.Replace(varValue, "$thoughts", string(cow.thoughts), -1)
				if newValue != varValue {
					colorVars[varName] = newValue
					madeChanges = true
				}
			}

			// Check for other variable references and resolve them
			for otherVar, otherVal := range colorVars {
				if strings.Contains(varValue, otherVar) && otherVar != varName {
					newValue := strings.Replace(varValue, otherVar, otherVal, -1)
					if newValue != varValue {
						colorVars[varName] = newValue
						madeChanges = true
						break // Break and restart the loop since we've modified a value
					}
				}
			}
		}
		if !madeChanges {
			break // No more replacements were made, we're done
		}
	}

	// Create base replacements
	replacements := []string{
		"\\\\", "\\",
		"\\@", "@",
		"\\$", "$",
		"\\e", "\033", // Add direct escape sequence replacement
		"$eyes", cow.eyes,
		"${eyes}", cow.eyes,
		"$tongue", cow.tongue,
		"${tongue}", cow.tongue,
		"$thoughts", string(cow.thoughts),
		"${thoughts}", string(cow.thoughts),
	}

	// Add color variable replacements
	for varName, varValue := range colorVars {
		replacements = append(replacements, varName, varValue)
	}

	r := strings.NewReplacer(replacements...)
	newsrc := r.Replace(string(src))
	separate := strings.Split(newsrc, "\n")
	mow := make([]string, 0, len(separate))
	cowStarted := false
	for _, line := range separate {
		// Check if we've reached the cow art
		if !cowStarted && strings.Contains(line, "<<EOC") {
			cowStarted = true
			continue
		}

		// End of cow art
		if strings.HasPrefix(line, "EOC") {
			break
		}

		// Only include lines from the actual cow art
		if cowStarted {
			mow = append(mow, line)
		}
	}
	return strings.Join(mow, "\n"), nil
}
