package maputil

func GetKeyByValue[K comparable, V comparable](m map[K]V, value V) (k K, found bool) {
	var zeroK K
	for k, v := range m {
		if v == value {
			return k, true
		}
	}
	return zeroK, false
}

func Keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, len(m))
	var i int
	for k := range m {
		keys[i] = k
		i++
	}
	return keys
}

func KeysBy[K comparable, V any, T any](m map[K]V, mapper func(item K) T) []T {
	keys := make([]T, 0, len(m))
	for k := range m {
		keys = append(keys, mapper(k))
	}
	return keys
}

func Filter[K comparable, V any](m map[K]V, predicate func(key K, value V) bool) map[K]V {
	result := make(map[K]V)
	for k, v := range m {
		if predicate(k, v) {
			result[k] = v
		}
	}
	return result
}

func MergeMaps(map1, map2 map[string]string) map[string]string {
	mergedMap := make(map[string]string)
	for key, value := range map1 {
		mergedMap[key] = value
	}
	for key, value := range map2 {
		mergedMap[key] = value
	}
	return mergedMap
}

func StringMapToInterfaceMap(stringMap map[string]string) map[string]interface{} {
	interfaceMap := make(map[string]interface{}, len(stringMap))
	for key, value := range stringMap {
		interfaceMap[key] = value
	}
	return interfaceMap
}
