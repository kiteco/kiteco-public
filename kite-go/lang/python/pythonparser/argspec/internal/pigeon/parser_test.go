package pigeon

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// valid cases from numpy/random/mtrand/mtrand.pyx
var numpyMtrandCases = []string{
	"seed(seed=None)\n",
	"get_state()\n",
	"set_state(state)\n",
	"random_sample(size=None)\n",
	"tomaxint(size=None)\n",
	"randint(low, high=None, size=None, dtype='l')\n",
	"bytes(length)\n",
	"choice(a, size=None, replace=True, p=None)\n",
	"uniform(low=0.0, high=1.0, size=None)\n",
	"rand(d0, d1, ..., dn)\n",
	"randn(d0, d1, ..., dn)\n",
	"random_integers(low, high=None, size=None)\n",
	"standard_normal(size=None)\n",
	"normal(loc=0.0, scale=1.0, size=None)\n",
	"beta(a, b, size=None)\n",
	"exponential(scale=1.0, size=None)\n",
	"standard_exponential(size=None)\n",
	"standard_gamma(shape, size=None)\n",
	"gamma(shape, scale=1.0, size=None)\n",
	"f(dfnum, dfden, size=None)\n",
	"noncentral_f(dfnum, dfden, nonc, size=None)\n",
	"chisquare(df, size=None)\n",
	"noncentral_chisquare(df, nonc, size=None)\n",
	"standard_cauchy(size=None)\n",
	"standard_t(df, size=None)\n",
	"vonmises(mu, kappa, size=None)\n",
	"pareto(a, size=None)\n",
	"weibull(a, size=None)\n",
	"power(a, size=None)\n",
	"laplace(loc=0.0, scale=1.0, size=None)\n",
	"gumbel(loc=0.0, scale=1.0, size=None)\n",
	"logistic(loc=0.0, scale=1.0, size=None)\n",
	"lognormal(mean=0.0, sigma=1.0, size=None)\n",
	"rayleigh(scale=1.0, size=None)\n",
	"wald(mean, scale, size=None)\n",
	"triangular(left, mode, right, size=None)\n",
	"binomial(n, p, size=None)\n",
	"negative_binomial(n, p, size=None)\n",
	"poisson(lam=1.0, size=None)\n",
	"zipf(a, size=None)\n",
	"geometric(p, size=None)\n",
	"hypergeometric(ngood, nbad, nsample, size=None)\n",
	"logseries(p, size=None)\n",
	"multivariate_normal(mean, cov[, size, check_valid, tol])\n",
	"multinomial(n, pvals, size=None)\n",
	"dirichlet(alpha, size=None)\n",
	"shuffle(x)\n",
	"permutation(x)\n",
}

func TestParseArgSpec(t *testing.T) {
	// NOTE: all cases must end with a newline, the exporter Parse
	// API will automatically add that final newline (because this
	// is a line-driven parser much like the epytext one).

	validCases := []string{
		"fn()\n",
		"\n\r\t\n   fn()\n",
		"fn.a()\n",
		"fn.a.b ()\n",
		"fn.a.b (  )\n",
		"\tfn.a.b ( x )\n",
		"\tfn.a.b (x,y) \n",
		"\tfn.a.b ( x , y ) \n",
		"\tfn.a.b ( x , y,z) \n",
		"\tfn(x=3)\n",
		"\tfn(x = 3)\n",
		"\tfn(x =None , y)\n",
		"\tfn(x = None , y=\"s\" )\n",
		"\tfn(x = true , y=false, ...)\n",
		"\tfn(x = true , y=false, ...,z)\n",
		"\tfn(x = true , y=false, ..., z= None)\n",
		"\tfn(x=foo.bar.baz, y = MaxValue, z = true)\n",
		"fn()\n\tThen Many Other Stuff.\nand()\nagain()\n.\n",
		"fn(x=\"a'b\")\n",
		"fn(x='a\"b')\n",
		"fn(*vararg)\n",
		"fn(**kwarg)\n",
		"fn(a=1,**kwarg)\n",
		"fn(a=1,*vararg)\n",
		"fn([a])\n",
		"fn(a[, b])\n",
		"fn(a=1[, b])\n",
		"fn([ a, b ])\n",
		"fn([ a, b ], c)\n",
		"fn( x, y) -> None\n",
		"fn(*)\n",
	}
	for _, c := range append(validCases, numpyMtrandCases...) {
		t.Run(c, func(t *testing.T) {
			_, err := Parse("", []byte(c))
			require.NoError(t, err, "%q", c)
		})
	}

	invalidCases := []string{
		"",
		" ",
		"\n",
		"fn\n",
		"...()\n",
		" fn(.)\n",
		" fn(..)\n",
		" fn(....)\n",
		" fn(... ,)\n",
		" fn(\n",
		" fn(  ).\n",
		" fn )\n",
		" fn. b()\n",
		" 1a()\n",
		" fn(a,)\n",
		" fn(a,\n",
		" fn(a\n",
		" fn a\n",
		" fn a)\n",
		" fn a, b\n",
		" fn a ()\n",
		" fn(x=)\n",
		" fn(x=1 =2)\n",
		" fn(x=1 2)\n",
		" fn(x=1 None True)\n",
		" fn(x=,y)\n",
		" fn(=)\n",
		"Some non argspec line.\n\tfn()\n",
		"fn(x=\"\"\")",
		"fn(x=''')",
		"fn(x=\"\n\")",
		"fn(x='\n')",
		"fn(**)\n",
		"fn(***x)\n",
		"fn(a**x)\n",
		"fn(*...)\n",
		"fn(**...)\n",
		"fn([]a)\n",
		"fn([a]])\n",
		"fn([[a])\n",
	}
	for _, c := range invalidCases {
		t.Run(c, func(t *testing.T) {
			_, err := Parse("", []byte(c))
			require.Error(t, err, "%q", c)
		})
	}
}
