package main

import (
	"log"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/manifest"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/pkg/errors"
)

type pathTransformer func(string) (string, error)

func manifestTransformPaths(m manifest.Manifest, f pathTransformer) (manifest.Manifest, error) {
	if len(m) == 0 {
		return nil, nil
	}

	newManifest := make(manifest.Manifest, len(m))
	for dist, lg := range m {
		// transform paths for LocatorGroup lg
		newLG := make(resources.LocatorGroup, len(lg))
		for name, l := range lg {
			// transform path for Locator l
			newPath, err := f(string(l))
			if err != nil {
				return nil, err
			}
			newLG[name] = resources.Locator(newPath)
		}

		newManifest[dist] = newLG
	}

	return newManifest, nil
}

func manifestCopyResources(src, dst manifest.Manifest) error {
	c := newCopyManager()
	for dist, lgSrc := range src {
		lgDst, exists := dst[dist]
		if !exists {
			return errors.Errorf("distribution %s does not exist in destination manifest", dist)
		}

		// copy lgSrc to lgDst
		for name, lSrc := range lgSrc {
			lDst, exists := lgDst[name]
			if !exists {
				return errors.Errorf("resource %s does not exist for distribution %s in destination manifest", name, dist)
			}

			err := c.copy(string(lSrc), string(lDst))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func manifestMerge(base, other manifest.Manifest) {
	if len(other) == 0 {
		return
	}
	if base == nil {
		log.Fatalln("cannot merge into nil Manifest")
	}

	for dist, lg := range other {
		oldLG, exists := base[dist]
		if !exists {
			oldLG = make(resources.LocatorGroup)
			base[dist] = oldLG
		}
		// merge lg into oldLG
		for name, l := range lg {
			oldLG[name] = l
		}
	}
}

func manifestExtract(m manifest.Manifest, dist keytypes.Distribution) (manifest.Manifest, error) {
	if lg, exists := m[dist]; exists {
		newM := make(manifest.Manifest, 1)
		newM[dist] = lg
		return newM, nil
	}
	return nil, errors.Errorf("distribution %s not found in manifest", dist)
}
