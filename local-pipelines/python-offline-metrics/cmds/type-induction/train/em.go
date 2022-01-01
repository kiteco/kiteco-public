package main

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"
)

// Joint probability P(a,t,f)
func (m *model) Joint(t *Type, f *Func, attrs []string) float64 {
	// P(a|t)
	pa := 1.
	for _, attr := range attrs {
		pa *= t.Attrs[attr]
	}

	// P(f) * P(t|f) * P(a|t)
	pr := f.Return[t]
	p := f.Prob * pr * pa
	logf(logLevelVerbose, "P(a=[%s],t=%s,f=%s) = P(f) * P(t|f) * P(a|t) = %f * %f * %f = %f\n",
		strings.Join(attrs, ","), t.Sym.Path().String(), f.Sym.Path().String(),
		f.Prob, pr, pa, p,
	)

	return p
}

// EStep updates the type distribution for each variable
func (m *model) EStep() {
	// compute P(t|a,f) for each variable,
	// note we have a separate distribution for each
	// variable
	for _, v := range m.variables {
		// reset parameters
		sum := v.Types.SetAll(variableSmoothing)

		// compute P(a,t,f) for each t,
		// we iterate over v.types so that we
		// only choose types which are consistent with
		// the set of attributes observed on the variable
		for t := range v.Types {
			p := m.Joint(t, v.Func, v.Attrs)
			sum += p

			// NOTE: add here so that we do not over write smoothing
			v.Types[t] += p
		}

		// compute P(t|a,f) via P(t|a,f) = P(a,t,f) / P(a,f)
		// and P(a,f) = sum_t(P(a,t,f))
		// safe to take normalize here because all examples
		// are consistent with atleast one type
		v.Types.Normalize(sum)
	}
}

// MStepFuncs updates the return distribution for each function
func (m *model) MStepFuncs() {
	for _, f := range m.funcs {
		// reset parameters
		sum := f.Return.SetAll(funcSmoothing)

		// compute new estimate for P(t|f);
		// note that if t were observed then this would just
		// be a counting problem, e.g the number of times f
		// returned a value of type t, however since t is not observed
		// we need to scale the counts by P(t|a,f)
		// NOTE: we explicitly iterate over f.Return so that we
		// only consider return types for the function that
		// are consistent with all attributes that we have
		// observed on the return value for that function
		for t := range f.Return {
			for _, usage := range f.Usages {
				// P(t|a,f)
				p := usage.Types[t]
				sum += p

				// NOTE: add here so that we do not over write smoothing
				f.Return[t] += p
			}
		}

		// normalize pseudo counts
		f.Return.Normalize(sum)
	}
}

// MStepTypes updates the attribute distribution for each Type
func (m *model) MStepTypes() {
	for _, t := range m.types {
		// reset parameters
		var sum float64
		for attr := range t.Attrs {
			sum += attrSmoothing
			t.Attrs[attr] = attrSmoothing
		}

		// compute new estimate for P(a|t); note that if
		// t were observed this would just be a counting problem,
		// e.g the number of times each attr occurred on a value of
		// type t, however since t is not observed we have to
		// scale the count by P(t|a,f), which for the M step
		// we consider as fixed
		for _, usage := range m.variables {
			if !consistentWithType(usage.Attrs, t) {
				continue
			}

			// P(t|a,f)
			p := usage.Types[t]
			for _, attr := range usage.Attrs {
				if _, ok := t.Attrs[attr]; ok {
					sum += p
					t.Attrs[attr] += p
				}
			}
		}

		// normalize pseudo counts
		invSum := 1. / sum
		for attr, p := range t.Attrs {
			t.Attrs[attr] = p * invSum
		}
	}
}

func (m *model) DatasetLogLoss() float64 {
	var logloss float64
	for _, v := range m.variables {
		// P(a_v,f_v) = sum_t(P(a_v,f_v,t_v))
		var p float64
		for t := range v.Types {
			// P(t_v,a_v,f_v)
			p += m.Joint(t, v.Func, v.Attrs)
		}

		logloss -= math.Log(p)
	}

	return logloss / float64(len(m.variables))
}

// EM algorithm for learning model parameters
func (m *model) EM(steps int, lossInterval int) {
	m.SanityCheck()

	fmt.Printf("starting EM with %d functions, %d types and %d variables\n",
		len(m.funcs), len(m.types), len(m.variables),
	)

	var lastLL float64
	start := time.Now()
	for step := 0; step < steps; step++ {
		if step%lossInterval == 0 {
			ll := m.DatasetLogLoss()
			fmt.Printf("step %d log loss %f took %v for %d steps\n",
				step, ll, time.Since(start), lossInterval,
			)

			if step > 0 && ll > lastLL {
				fmt.Println("WARNING: log loss has increased")
			}

			if math.Abs(ll-lastLL) < eps {
				fmt.Printf("training has converged after %d steps with dataset log loss %f\n", step-1, ll)
				break
			}
			start = time.Now()
			lastLL = ll
		}
		m.EStep()
		m.MStepFuncs()
		m.MStepTypes()
	}
}

func (m *model) SanityCheck() {
	for _, v := range m.variables {
		if len(v.Types) == 0 {
			log.Fatalf("no types for variable %s\n", v.String())
		}

		for t := range v.Types {
			if !consistentWithType(v.Attrs, t) {
				log.Fatalf("variable %s is not consistent with type %s\n", v.String(), t.String())
			}
		}
	}

	for _, f := range m.funcs {
		if len(f.Return) == 0 {
			log.Fatalf("no return types for func %s\n", f.String())
		}
		for t := range f.Return {
			var consistent bool
			for _, usage := range f.Usages {
				if consistentWithType(usage.Attrs, t) {
					consistent = true
					break
				}
			}

			if !consistent {
				log.Fatalf("for func %s no return types are consistent with the %d usages\n",
					f.String(), len(f.Usages))
			}
		}
	}
}
