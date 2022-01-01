package recommend

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileIndex(t *testing.T) {
	r := recommender{
		opts: Options{
			Root: testDir,
		},
	}
	f := r.newFileIndex()

	alphaOriginal := testDir.Join("alpha")
	betaGammaOriginal := testDir.Join("beta", "gamma")
	deltaOriginal := testDir.Join("delta")

	alphaID1, err := f.toID(alphaOriginal)
	require.NoError(t, err)
	betaGammaID, err := f.toID(betaGammaOriginal)
	require.NoError(t, err)
	deltaID, err := f.toID(deltaOriginal)
	require.NoError(t, err)
	alphaID2, err := f.toID(alphaOriginal)
	require.NoError(t, err)

	require.Equal(t, fileID(0), alphaID1)
	require.Equal(t, fileID(1), betaGammaID)
	require.Equal(t, fileID(2), deltaID)
	require.Equal(t, fileID(0), alphaID2)

	alpha1, err := f.fromID(alphaID1)
	require.NoError(t, err)
	betaGamma, err := f.fromID(betaGammaID)
	require.NoError(t, err)
	delta, err := f.fromID(deltaID)
	require.NoError(t, err)
	alpha2, err := f.fromID(alphaID2)
	require.NoError(t, err)

	require.Equal(t, alphaOriginal, alpha1)
	require.Equal(t, betaGammaOriginal, betaGamma)
	require.Equal(t, deltaOriginal, delta)
	require.Equal(t, alphaOriginal, alpha2)
}
