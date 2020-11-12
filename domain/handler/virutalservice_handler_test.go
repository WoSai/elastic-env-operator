package handler

import (
	"github.com/wosai/elastic-env-operator/domain/entity"
	"github.com/wosai/elastic-env-operator/domain/util"
	"gotest.tools/assert"
	"testing"
)

func TestGeneratePlaneHttpRoute (t *testing.T)  {
	host := "host1"
	plane := "plane1"
	path := "/v1"
	route := generatePlaneHttpRoute(host, plane, path)
	assert.Equal(t, len(route.Route), 1)
	assert.Equal(t, route.Route[0].Destination.Host, "host1")
	assert.Equal(t, route.Route[0].Destination.Subset, util.GetSubsetName(host, plane))
	assert.Equal(t, len(route.Match), 3)
	assert.Equal(t, route.Match[0].Headers[entity.XEnvFlag].GetExact(), plane)
	assert.Equal(t, route.Match[0].Uri.GetPrefix(), path)
	assert.Equal(t, route.Match[1].QueryParams[entity.XEnvFlag].GetExact(), plane)
	assert.Equal(t, route.Match[1].Uri.GetPrefix(), path)
	assert.Equal(t, route.Match[2].SourceLabels[entity.PlaneKey], plane)
	assert.Equal(t, route.Match[2].Uri.GetPrefix(), path)

	host = "host1"
	plane = "plane1"
	path = "/"
	route = generatePlaneHttpRoute(host, plane, path)
	assert.Equal(t, len(route.Route), 1)
	assert.Equal(t, route.Route[0].Destination.Host, host)
	assert.Equal(t, route.Route[0].Destination.Subset, util.GetSubsetName(host, plane))
	assert.Equal(t, len(route.Match), 3)
	assert.Equal(t, route.Match[0].Headers[entity.XEnvFlag].GetExact(), plane)
	assert.Equal(t, route.Match[0].Uri.GetPrefix(), "")
	assert.Equal(t, route.Match[1].QueryParams[entity.XEnvFlag].GetExact(), plane)
	assert.Equal(t, route.Match[1].Uri.GetPrefix(), "")
	assert.Equal(t, route.Match[2].SourceLabels[entity.PlaneKey], plane)
	assert.Equal(t, route.Match[2].Uri.GetPrefix(), "")
}

func TestGenerateBaseHttpRoute (t *testing.T)  {
	host := "host1"
	path := "/v1"
	route := generateBaseHttpRoute(host, path)
	assert.Equal(t, len(route.Route), 1)
	assert.Equal(t, route.Route[0].Destination.Host, host)
	assert.Equal(t, route.Route[0].Destination.Subset, util.GetSubsetName(host, "base"))
	assert.Equal(t, len(route.Match), 1)
	assert.Equal(t, route.Match[0].Uri.GetPrefix(), path)
	assert.Equal(t, len(route.Match[0].Headers), 0)
	assert.Equal(t, len(route.Match[0].QueryParams), 0)
	assert.Equal(t, len(route.Match[0].SourceLabels), 0)

	host = "host1"
	path = "/"
	route = generateBaseHttpRoute(host, path)
	assert.Equal(t, len(route.Route), 1)
	assert.Equal(t, route.Route[0].Destination.Host, host)
	assert.Equal(t, route.Route[0].Destination.Subset, util.GetSubsetName(host, "base"))
	assert.Equal(t, len(route.Match), 0)
}
