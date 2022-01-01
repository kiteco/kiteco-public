package codebase

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/navigation/git"
	"github.com/kiteco/kiteco/kite-go/navigation/ignore"
	"github.com/kiteco/kiteco/kite-go/navigation/recommend"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/require"
)

type loadTC struct {
	errBefore         error
	ignorerBefore     ignore.Ignorer
	recommenderBefore recommend.Recommender
	statusBefore      ProjectStatus
	statusAfter       ProjectStatus
	loadError         error
	recommenderNotNil bool
}

func TestMaybeLoad(t *testing.T) {
	tcs := []loadTC{
		loadTC{
			statusBefore:      Inactive,
			statusAfter:       Active,
			recommenderNotNil: true,
		},
		loadTC{
			statusBefore: Inactive,
			statusAfter:  Failed,
			loadError:    errors.New("failed to load"),
		},
		loadTC{
			statusBefore: InProgress,
			statusAfter:  InProgress,
		},
		loadTC{
			recommenderBefore: mockRecommender{},
			statusBefore:      Active,
			statusAfter:       Active,
			recommenderNotNil: true,
		},
		loadTC{
			errBefore:     errors.New("failed to load"),
			ignorerBefore: mockIgnorer{},
			statusBefore:  Failed,
			statusAfter:   Failed,
		},
		loadTC{
			errBefore:     errors.New("failed to load"),
			ignorerBefore: mockIgnorer{shouldRebuild: true},
			statusBefore:  Failed,
			statusAfter:   Failed,
		},
		loadTC{
			errBefore:         recommend.ErrOpenedTooManyFiles,
			ignorerBefore:     mockIgnorer{shouldRebuild: true},
			statusBefore:      Failed,
			statusAfter:       Active,
			recommenderNotNil: true,
		},
	}
	for _, tc := range tcs {
		project := &projectNavigator{
			state: projectState{
				status:      tc.statusBefore,
				err:         tc.errBefore,
				ignorer:     tc.ignorerBefore,
				recommender: tc.recommenderBefore,
			},
			load: func(ctx kitectx.Context, s git.Storage, ignoreOpts ignore.Options, recOpts recommend.Options) projectState {
				if tc.loadError != nil {
					return projectState{
						status: Failed,
						err:    tc.loadError,
					}
				}
				return projectState{
					status:      Active,
					recommender: mockRecommender{},
				}
			},
			m: new(sync.Mutex),
		}
		s, err := git.NewStorage(git.StorageOptions{})
		require.NoError(t, err)
		project.maybeLoad(kitectx.Background(), s, 1e6, 1e5)
		time.Sleep(100 * time.Millisecond)

		require.Equal(t, tc.statusAfter, project.state.status)
		require.Equal(t, tc.recommenderNotNil, project.state.recommender != nil)
	}
}

type projectNavigateTC struct {
	pageIndex     int
	expected      []recommend.File
	expectedError error
}

func TestProjectNavigate(t *testing.T) {
	project := projectNavigator{
		state: projectState{
			status: Active,
			recommender: mockRecommender{
				files: []recommend.File{
					{Path: "alpha"},
					{Path: "beta"},
					{Path: "gamma"},
				},
			},
		},
		m: new(sync.Mutex),
	}
	iter, err := project.navigate(kitectx.Background(), recommend.Request{})
	require.NoError(t, err)
	alphabeta, err := iter.Next(2)
	require.NoError(t, err)
	require.Equal(t, []recommend.File{{Path: "alpha"}, {Path: "beta"}}, alphabeta)
	gamma, err := iter.Next(2)
	require.NoError(t, err)
	require.Equal(t, []recommend.File{{Path: "gamma"}}, gamma)
	empty, err := iter.Next(2)
	require.Equal(t, err, ErrEmptyIterator)
	require.Nil(t, empty)
}

type testProjectNavigateNotActiveTC struct {
	state         projectState
	expectedError error
}

var errForTesting = errors.New("for testing")

func TestProjectNavigateNotActive(t *testing.T) {
	tcs := []testProjectNavigateNotActiveTC{
		testProjectNavigateNotActiveTC{
			state: projectState{
				status: Inactive,
			},
			expectedError: ErrShouldLoad,
		},
		testProjectNavigateNotActiveTC{
			state: projectState{
				status: InProgress,
			},
			expectedError: errWasInProgress,
		},
		testProjectNavigateNotActiveTC{
			state: projectState{
				status: Failed,
				err:    errForTesting,
			},
			expectedError: errForTesting,
		},
		testProjectNavigateNotActiveTC{
			state: projectState{
				status: IgnorerFailed,
				err:    errForTesting,
			},
			expectedError: errForTesting,
		},
	}

	for _, tc := range tcs {
		project := projectNavigator{
			state: tc.state,
			m:     new(sync.Mutex),
		}
		_, err := project.navigate(kitectx.Background(), recommend.Request{})
		require.Equal(t, tc.expectedError, err)
	}
}
