package resources

import (
	"io/ioutil"
	"math/rand"
	"reflect"
	"testing/quick"
)

func locatorOrPanic(r Resource) Locator {
	f, err := ioutil.TempFile("", "kitetest")
	if err != nil {
		panic(err)
	}

	err = r.Encode(f)
	if err != nil {
		panic(err)
	}

	err = f.Close()
	if err != nil {
		panic(err)
	}

	return Locator(f.Name())
}

func lgFromRG(rg Group) LocatorGroup {
	return lgFromRGVal(reflect.ValueOf(rg))
}

func lgFromRGVal(rgVal reflect.Value) LocatorGroup {
	rgTy := rgVal.Type()

	l := make(LocatorGroup)
	for i := 0; i < rgTy.NumField(); i++ {
		l[rgTy.Field(i).Name] = locatorOrPanic(rgVal.Field(i).Interface().(Resource))
	}

	return l
}

// Generate implements quick.Generator for generating random LocatorGroups for testing
func (LocatorGroup) Generate(rand *rand.Rand, size int) reflect.Value {
	rgTy := reflect.TypeOf((*Group)(nil)).Elem()
	rgVal, ok := quick.Value(rgTy, rand)
	if !ok {
		panic("failed to generate a random resources Group: this should never happen")
	}

	return reflect.ValueOf(lgFromRGVal(rgVal))
}
