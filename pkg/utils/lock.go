package utils

import (
	"log"
	"reflect"
	"sync"

	"istio.io/api/networking/v1beta1"
)

//Operations on slices are not thread safe, so this is to lock the slice to stop the
//possibility of concurrent writes
type SliceLock struct {
	*sync.Mutex
}

func NewLock() SliceLock {
	var mutex sync.Mutex
	return SliceLock{
		Mutex: &mutex,
	}
}

func (sl *SliceLock) Append(orig, appArr []*v1beta1.HTTPRoute) []*v1beta1.HTTPRoute {

	var meshHttpRoutes []*v1beta1.HTTPRoute

	for _, v := range appArr {
		if Contains(orig, v) {
			log.Printf("Route %s already exists, not adding", v.Name)
		} else {
			meshHttpRoutes = append(meshHttpRoutes, v)
		}
	}

	sl.Lock()
	defer sl.Unlock()
	return append(orig, appArr...)
}

func (sl *SliceLock) Delete(orig, key []*v1beta1.HTTPRoute) []*v1beta1.HTTPRoute {
	sl.Lock()
	defer sl.Unlock()
	for i := range key {
		for j := range orig {
			if reflect.DeepEqual(orig[j], key[i]) {
				copy(orig[j:], orig[j+1:])
				orig[len(orig)-1] = nil
				orig = orig[:len(orig)-1]
				break
			}
		}
	}

	return orig
}

func (sl *SliceLock) Update(orig, newArr, oldArr []*v1beta1.HTTPRoute) []*v1beta1.HTTPRoute {
	var meshHttpRoutes []*v1beta1.HTTPRoute

	sl.Lock()
	defer sl.Unlock()

	//Delete old routes
	for i := range oldArr {
		for j := range orig {
			if reflect.DeepEqual(orig[j], oldArr[i]) {
				copy(orig[j:], orig[j+1:])
				orig[len(orig)-1] = nil
				orig = orig[:len(orig)-1]
				break
			}
		}
	}

	meshHttpRoutes = append(orig, newArr...)
	return meshHttpRoutes
}
