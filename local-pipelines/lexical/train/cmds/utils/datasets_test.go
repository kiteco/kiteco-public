package utils

import (
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCStyle(t *testing.T) {
	for _, dt := range []DatasetType{TrainDataset, ValidateDataset, TestDataset} {
		ds := DatasetForLang(dt, lexicalv0.CStyleGroup)
		require.Len(t, ds, 7)
		for _, l := range lexicalv0.CStyleGroup.Langs {
			for _, ext := range lang.LanguageTags[l].Exts {
				var ok bool
				for _, d := range ds {
					if strings.HasSuffix(d, ext+"/"+string(dt)) {
						ok = true
						break
					}
				}
				assert.True(t, ok, "missing dataset for lang %s and ext %s", l.Name(), ext)
			}
		}
	}
}

func TestJavaPlusPlus(t *testing.T) {
	for _, dt := range []DatasetType{TrainDataset, ValidateDataset, TestDataset} {
		ds := DatasetForLang(dt, lexicalv0.JavaPlusPlusGroup)
		require.Len(t, ds, 3)
		for i, d := range ds {
			l := lexicalv0.JavaPlusPlusGroup.Langs[i]
			ext := lang.LanguageTags[l].Ext
			assert.True(t, strings.HasSuffix(d, ext+"/"+string(dt)), "lang %s, ext %s, path %s", l.Name(), ext, d)
		}
	}
}

func TestWeb(t *testing.T) {
	for _, dt := range []DatasetType{TrainDataset, ValidateDataset, TestDataset} {
		ds := DatasetForLang(dt, lexicalv0.WebGroup)
		require.Len(t, ds, 8)
		for i, d := range ds {
			l := lexicalv0.WebGroup.Langs[i]
			ext := lang.LanguageTags[l].Ext
			assert.True(t, strings.Contains(d, ext), "lang %s, ext %s, path %s", l.Name(), ext, d)
		}
	}
}

func TestAllLangs(t *testing.T) {
	for _, dt := range []DatasetType{TrainDataset, ValidateDataset, TestDataset} {
		ds := DatasetForLang(dt, lexicalv0.AllLangsGroup)
		numOldDatasets := 5 // go,py,jsx,js,vue
		numNewDatasets := 18
		require.Len(t, ds, numOldDatasets+numNewDatasets)
		for _, l := range lexicalv0.AllLangsGroup.Langs {
			for _, ext := range lang.LanguageTags[l].Exts {
				if ext == "pyw" || ext == "pyt" {
					// skip pyw and pyt extension for tests
					continue
				}

				var ok bool
				for _, d := range ds {
					if strings.HasSuffix(d, ext+"/"+string(dt)) {
						// new dataset
						ok = true
						break
					}
					if strings.Contains(d, ext) && strings.Contains(d, string(dt)) {
						// old dataset
						ok = true
						break
					}
				}
				assert.True(t, ok, "missing dataset for lang %s and ext %s", l.Name(), ext)
			}
		}
	}
}

func TestMiscLangs(t *testing.T) {
	for _, dt := range []DatasetType{TrainDataset, ValidateDataset, TestDataset} {
		ds := DatasetForLang(dt, lexicalv0.MiscLangsGroup)
		numOldDatasets := 2 // go,py
		numNewDatasets := 3
		require.Len(t, ds, numOldDatasets+numNewDatasets)
		for _, l := range lexicalv0.MiscLangsGroup.Langs {
			for _, ext := range lang.LanguageTags[l].Exts {
				if ext == "pyw" || ext == "pyt" {
					// skip pyw and pyt extension for tests
					continue
				}

				var ok bool
				for _, d := range ds {
					if strings.HasSuffix(d, ext+"/"+string(dt)) {
						// new dataset
						ok = true
						break
					}
					if strings.Contains(d, ext) && strings.Contains(d, string(dt)) {
						// old dataset
						ok = true
						break
					}
				}
				assert.True(t, ok, "missing dataset for lang %s and ext %s", l.Name(), ext)
			}
		}
	}
}
