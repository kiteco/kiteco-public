package helpers

import (
	"os"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/errors"
	yaml "gopkg.in/yaml.v2"
)

// InstallMeta specifies how to download/install a package
type InstallMeta struct {
	Manager string `yaml:"manager"`
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

// DistributionMeta encapsulates information needed to acquire, install, and analyze a logical (pythonresource) distribution
type DistributionMeta struct {
	Name    string        `yaml:"name"`
	Python  string        `yaml:"python"`
	Version string        `yaml:"version"`
	Install []InstallMeta `yaml:"install"`
	Depends []InstallMeta `yaml:"extra_depends"`
}

func (m DistributionMeta) validate() error {
	if m.Name == "" {
		return errors.New("distribution must have name")
	}
	switch m.Python {
	case "python3":
	default:
		return errors.New("invalid python version %s (must be python3)", m.Python)
	}

	pipSet := make(map[string]struct{})
	aptSet := make(map[string]struct{})

	validateInstall := func(t InstallMeta) error {
		var set map[string]struct{}
		switch t.Manager {
		case "pip":
			set = pipSet
		case "apt":
			set = aptSet
		default:
			return errors.Errorf("invalid manager specification %s for %s (must be apt or pip)", t.Manager, t.Name)
		}

		if _, ok := set[t.Name]; ok {
			return errors.Errorf("install specification %s (%s) mentioned twice", t.Name, t.Manager)
		}
		set[t.Name] = struct{}{}
		return nil
	}

	for i, t := range m.Install {
		if t.Name == "" {
			t.Name = m.Name
			if t.Version != "" {
				return errors.Errorf("install specification with version %s must have name", t.Version)
			}
			t.Version = m.Version
			m.Install[i] = t
		}

		if err := validateInstall(t); err != nil {
			return err
		}
	}

	for _, t := range m.Depends {
		if t.Name == "" {
			return errors.Errorf("dependency specification must have name")
		}

		if err := validateInstall(t); err != nil {
			return err
		}
	}

	if m.Name != "builtin-stdlib" && len(pipSet) == 0 && len(aptSet) == 0 {
		return errors.Errorf("no install/dependecy specification")
	}

	return nil
}

func validateDistributions(dists []DistributionMeta) error {
	kdistSet := make(map[keytypes.Distribution]struct{})
	for _, dist := range dists {
		kdist := keytypes.Distribution{Name: dist.Name, Version: dist.Version}
		if _, ok := kdistSet[kdist]; ok {
			return errors.Errorf("duplicate distribution %s", kdist)
		}
		kdistSet[kdist] = struct{}{}
		if err := dist.validate(); err != nil {
			return errors.Wrapf(err, "invalid distribution %s", kdist)
		}
	}
	return nil
}

// LoadDistributions loads distribution metadata from distributions.yaml
func LoadDistributions(fname string) ([]DistributionMeta, error) {
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var dists []DistributionMeta
	if err := yaml.NewDecoder(f).Decode(&dists); err != nil {
		return nil, err
	}
	if err := validateDistributions(dists); err != nil {
		return nil, err
	}

	return dists, nil
}
