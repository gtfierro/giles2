package views

func byteSliceMatch(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, va := range a {
		if va != b[i] {
			return false
		}
	}
	return true
}

func String(s string) *string {
	return &s
}
