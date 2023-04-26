package search

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tarantool/tt/cli/version"
)

type getVersionsInputValue struct {
	data   *map[string]interface{}
	filter SearchFlags
}

type getVersionsOutputValue struct {
	result BundleInfoSlice
	err    error
}

func TestGetVersions(t *testing.T) {
	assert := assert.New(t)
	pkgRelease := "tarantool-enterprise-sdk-nogc64-2.10.6-0-r549.linux.x86_64.tar.gz"
	pkgDebug := "tarantool-enterprise-sdk-debug-nogc64-2.10.6-0-r549.linux.x86_64.tar.gz"

	testCases := make(map[getVersionsInputValue]getVersionsOutputValue)

	inputData0 := map[string]interface{}{
		"random": "data",
	}

	testCases[getVersionsInputValue{data: &inputData0, filter: SearchAll}] =
		getVersionsOutputValue{
			result: nil,
			err:    fmt.Errorf("no packages found for this OS or release version"),
		}

	inputData1 := map[string]interface{}{
		"2.10": []interface{}{pkgRelease},
	}

	testCases[getVersionsInputValue{data: &inputData1, filter: SearchRelease}] =
		getVersionsOutputValue{
			result: []BundleInfo{
				{
					Version: version.Version{
						Major:      2,
						Minor:      10,
						Patch:      6,
						Additional: 0,
						Revision:   549,
						Release:    version.Release{Type: version.TypeRelease},
						Hash:       "",
						Str:        "nogc64-2.10.6-0-r549",
						Tarball:    pkgRelease,
						BuildName:  "nogc64",
					},
					Release: "2.10",
					Package: "enterprise",
				},
			},
			err: nil,
		}

	inputData2 := map[string]interface{}{
		"2.10": []interface{}{pkgRelease},
	}

	testCases[getVersionsInputValue{data: &inputData2, filter: SearchDebug}] =
		getVersionsOutputValue{
			result: nil,
			err:    fmt.Errorf("no packages found for this OS or release version"),
		}

	inputData3 := map[string]interface{}{
		"2.10": []interface{}{pkgDebug},
	}

	testCases[getVersionsInputValue{data: &inputData3, filter: SearchDebug}] =
		getVersionsOutputValue{
			result: []BundleInfo{
				{
					Version: version.Version{
						Major:      2,
						Minor:      10,
						Patch:      6,
						Additional: 0,
						Revision:   549,
						Release:    version.Release{Type: version.TypeRelease},
						Hash:       "",
						Str:        "debug-nogc64-2.10.6-0-r549",
						Tarball:    pkgDebug,
						BuildName:  "debug-nogc64",
					},
					Release: "2.10",
					Package: "enterprise",
				},
			},
			err: nil,
		}

	inputData4 := map[string]interface{}{
		"2.10": []interface{}{pkgDebug},
	}

	testCases[getVersionsInputValue{data: &inputData4, filter: SearchRelease}] =
		getVersionsOutputValue{
			result: nil,
			err:    fmt.Errorf("no packages found for this OS or release version"),
		}

	for input, output := range testCases {
		versions, err := getBundles(*input.data, input.filter)

		if output.err == nil {
			assert.Nil(err)
			assert.Equal(output.result, versions)
		} else {
			assert.Equal(output.err, err)
		}
	}
}
