package strings

func ContainsIn(s string, set []string) bool {
	for _, t := range set {
		if s == t {
			return true
		}
	}
	return false
}
