package utils

func StringsContain(ss []string, s string) bool {
	for _, s2 := range ss {
		if s2 == s {
			return true
		}
	}

	return false
}
