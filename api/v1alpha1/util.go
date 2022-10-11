package v1alpha1

func MergeStringMap(base, toMerge map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range toMerge {
		result[k] = v
	}
	return result
}
