package ranking

import (
	"log"
	"math"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadClassifier(t *testing.T) {
	buf, err := os.Open("testdata/logistmodel.json")
	if err != nil {
		log.Fatal(err)
	}
	_, err = NewBinaryClassifierFromJSON(buf)
	assert.NoError(t, err, "")
}

func TestLogisticRegressionEvaluate(t *testing.T) {
	buf, err := os.Open("testdata/logistmodel.json")
	if err != nil {
		log.Fatal(err)
	}
	clf, _ := NewBinaryClassifierFromJSON(buf)

	feat := []float64{1, 1, 0, 0, 1, 0, 0.47003870527410735, 0.16566072434470192, 0.35533923072115275, -2.5471351639196325}
	act := clf.Scorer.Evaluate(feat)
	exp := 0.99842178
	if math.Abs(act-exp) > 1e-6 {
		t.Errorf("expected %f, but got %f\n", exp, act)
	}

	feat = []float64{0, 0, 0, 0, 0, 1, 0.00008076823309974973, 0.002006891671968735, 0.01705925858080295, -2.132830450202419}
	act = clf.Scorer.Evaluate(feat)
	exp = 0.07226872
	if math.Abs(act-exp) > 1e-6 {
		t.Errorf("expected %f, but got %f\n", exp, act)
	}
}
