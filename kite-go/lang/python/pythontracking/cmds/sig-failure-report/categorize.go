package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"path"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-go/localfiles/offlineconf"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
)

// - categories

type prefixResolves struct {
	Global bool   `json:"global"`
	Prefix string `json:"prefix"`
	To     string `json:"to"`
}

type unresolvedName struct {
	Name        string `json:"name"`
	Importable  bool   `json:"importable"`
	CheckedSite bool   `json:"checked_site"`
	InSite      bool   `json:"in_site"`
}

// exactly one field should be non-zero:
type category struct {
	Resolves       bool
	PrefixResolves *prefixResolves
	UnresolvedName *unresolvedName
	InBadNode      bool
}

// MarshalJSON implements json.Marshaler, splitting out a category into a "name" and optional "data"
func (c *category) MarshalJSON() ([]byte, error) {
	var ser struct {
		Name string      `json:"name"`
		Data interface{} `json:"data,omitempty"`
	}

	switch {
	case c == nil:
		ser.Name = "other"
	case c.Resolves:
		ser.Name = "resolves"
	case c.PrefixResolves != nil:
		ser.Name = "prefix_resolves"
		ser.Data = c.PrefixResolves
	case c.UnresolvedName != nil:
		ser.Name = "unresolved_name"
		ser.Data = c.UnresolvedName
	case c.InBadNode:
		ser.Name = "in_bad_node"
	default:
		panic("unexpected category")
	}

	return json.Marshal(ser)
}

// - categorization

func (a analyzer) categorize(id analyze.MessageID, track *pythontracking.Event, ctx *python.Context) *category {
	switch track.Failure() {
	case string(pythontracking.UnresolvedValueFailure):
		return a.catUnresolved(id, track, ctx)
	}
	return nil
}

func (a analyzer) catUnresolved(id analyze.MessageID, track *pythontracking.Event, ctx *python.Context) *category {
	callExpr, _, inBad := python.FindCallExpr(kitectx.Background(), ctx.AST, ctx.Buffer, ctx.Cursor)
	if callExpr == nil {
		log.Printf("[ERROR] python.FindCallExpr returned nil for an unresolved_value case")
		return nil
	}
	// assume callExpr != nil and panic otherwise, since we're in an unresolved value case
	fExpr := callExpr.Func
	if ref := ctx.Resolved.References[fExpr]; ref != nil {
		return &category{Resolves: true}
	}

	cur := fExpr
	for i := 0; i < 100; i++ {
		if ref := ctx.Resolved.References[cur]; ref != nil {
			resolvedValue, isGlobal := getValueName(ref, ctx.Importer.Global)
			return &category{
				PrefixResolves: &prefixResolves{
					Prefix: string(ctx.Buffer[cur.Begin():cur.End()]),
					To:     resolvedValue,
					Global: isGlobal,
				},
			}
		}
		switch e := cur.(type) {
		case *pythonast.AttributeExpr:
			cur = e.Value
		default:
			break
		}
	}

	if name, ok := cur.(*pythonast.NameExpr); ok {
		lit := name.Ident.Literal
		if lit == "" {
			if inBad {
				return &category{InBadNode: true}
			}
			log.Printf("[ERROR] empty nameexpr %s", id)
		}

		res := &category{UnresolvedName: &unresolvedName{Name: lit}}
		if v, ok := ctx.Importer.ImportAbs(kitectx.Background(), lit); ok && v != nil {
			res.UnresolvedName.Importable = true
		} else if rand.Intn(100) < sitePackagesCheckPct && a.inSitePackages(lit, track.Region, ctx.User, ctx.Machine) {
			res.UnresolvedName.InSite = true
		}
		return res
	}

	if inBad {
		return &category{InBadNode: true}
	}

	return nil
}

// - helpers

func (a analyzer) inSitePackages(name string, region string, user int64, machine string) bool {
	manager := offlineconf.GetFileManager(region)
	if manager == nil {
		return false
	}

	files, err := manager.List(user, machine)
	if err != nil {
		return false
	}

	for _, f := range files {
		if f == nil {
			continue
		}

		modOrPkg := ""
		curPath := strings.TrimSuffix(f.Name, "/")
		for curPath != "" {
			dir, file := path.Split(curPath)
			if file == "site-packages" {
				if modOrPkg == name {
					return true
				}
				break // we've already found site-packages, no need to continue
			}
			modOrPkg = strings.TrimSuffix(file, ".py")
			curPath = strings.TrimSuffix(dir, "/")
		}
	}
	return false
}
