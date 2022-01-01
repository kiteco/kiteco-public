package pythonmetrics

import (
	"fmt"
	"go/token"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
)

// ProcessFileOffline processes a python file and generates all the reference comparisons for all references
// contained in the referenceMap. The offline mode includes text from the python file and should NOT be used by kite local.
func ProcessFileOffline(filename string, rast *pythonanalyzer.ResolvedAST, referenceMap ReferenceMap) ([]ReferenceComparison, map[pythonast.Expr]ReferenceResolutionComparison, error) {
	nodeMap := make(map[pythonast.Expr]ReferenceResolutionComparison, len(referenceMap))
	var result []ReferenceComparison
	for expr := range rast.References {
		ref, err := extractReference(rast, referenceMap, expr, filename, nodeMap, true)
		if err != nil {
			return nil, nil, err
		}
		if ref != nil {
			result = append(result, *ref)
		}
	}
	return result, nodeMap, nil
}

// ProcessFileOnline processes a python file and generates all the reference comparisons for all references
// contained in the referenceMap; all fields containing any user data (e.g user code or user types) are not included.
func ProcessFileOnline(filename string, rast *pythonanalyzer.ResolvedAST, referenceMap ReferenceMap) ([]ReferenceComparisonOnline, error) {
	var result []ReferenceComparisonOnline
	for expr := range rast.References {
		ref, err := extractReference(rast, referenceMap, expr, filename, nil, false)
		if err != nil {
			return nil, err
		}
		if ref != nil {
			result = append(result, ref.OnlineFields)
		}
	}
	return result, nil
}

func refKey(start, end token.Pos) string {
	return fmt.Sprintf("%d;%d", start, end)
}

func extractReference(rast *pythonanalyzer.ResolvedAST, refMap ReferenceMap, expr pythonast.Expr, filename string, nodeMap map[pythonast.Expr]ReferenceResolutionComparison, offlineMode bool) (*ReferenceComparison, error) {
	var symbol *pythontype.Symbol
	var val pythontype.Value

	if name, ok := expr.(*pythonast.NameExpr); ok {
		table, _ := rast.TableAndScope(name)
		if table != nil {
			symbol = table.Find(name.Ident.Literal)
			if symbol != nil {
				val = symbol.Value
			}
		}
	}
	if val == nil {
		val = rast.References[expr]
	}

	kiteResolutionLevel := Unknown
	if val != nil {
		kiteResolutionLevel = Known
		if _, ok := val.(pythontype.Union); ok {
			kiteResolutionLevel = UnionType
		}
	}

	intelliJRef := refMap[refKey(expr.Begin(), expr.End())]

	if intelliJRef != nil {
		intelliJResolutionLevel, err := newResolutionLevel(intelliJRef.ReferenceType.ResolutionLevel)
		if err != nil {
			return nil, err
		}
		if offlineMode {
			nodeMap[expr] = ReferenceResolutionComparison{intelliJResolutionLevel, kiteResolutionLevel}
		}
		onlineFields := ReferenceComparisonOnline{
			IntelliJSymbolResolved:  intelliJRef.Resolved,
			IntelliJResolutionLevel: intelliJResolutionLevel,
			IntelliJTypeOfType:      intelliJRef.ReferenceType.TypeOfType,
			KiteSymbolResolved:      symbol != nil,
			KiteResolutionLevel:     kiteResolutionLevel,
			KiteASTType:             fmt.Sprintf("%T", expr),
			KiteValueType:           fmt.Sprintf("%T", val),
		}

		result := ReferenceComparison{OnlineFields: onlineFields}
		if offlineMode {
			result.Text = intelliJRef.Text
			result.Begin = intelliJRef.Start
			result.End = intelliJRef.End
			result.KiteValue = fmt.Sprintf("%v", val)
			result.KiteSymbol = fmt.Sprintf("%v", symbol)
			result.IntelliJType = intelliJRef.ReferenceType.Type
			result.Filename = filename
		}

		intelliJRef.FoundInKite = true
		return &result, nil
	}
	return nil, nil
}
