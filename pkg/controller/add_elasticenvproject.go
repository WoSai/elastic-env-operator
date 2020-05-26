package controller

import (
	"github.com/wosai/elastic-env-operator/pkg/controller/elasticenvproject"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, elasticenvproject.Add)
}
