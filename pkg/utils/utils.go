package utils

import (
	"reflect"

	"istio.io/api/networking/v1beta1"
)

func Contains(orig []*v1beta1.HTTPRoute, elem *v1beta1.HTTPRoute) bool {
	for _, v := range orig {
		if reflect.DeepEqual(v, elem) {
			return true
		}
	}

	return false
}