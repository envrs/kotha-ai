package prompt

// Rune-level cursor movement helpers for the composer buffer.

func isWordChar(r rune) bool {
	return r == '_' || r == '-' ||
		(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9')
}

// PrevWordStart returns the cursor offset of the previous word start.
func PrevWordStart(buf []rune, pos int) int {
	if pos > len(buf) {
		pos = len(buf)
	}
	i := pos
	for i > 0 && !isWordChar(buf[i-1]) {
		i--
	}
	for i > 0 && isWordChar(buf[i-1]) {
		i--
	}
	return i
}

// NextWordEnd returns the cursor offset just past the next word end.
func NextWordEnd(buf []rune, pos int) int {
	if pos < 0 {
		pos = 0
	}
	i := pos
	for i < len(buf) && !isWordChar(buf[i]) {
		i++
	}
	for i < len(buf) && isWordChar(buf[i]) {
		i++
	}
	return i
}

// LineStart returns the offset of the current line start.
func LineStart(buf []rune, pos int) int {
	if pos > len(buf) {
		pos = len(buf)
	}
	for i := pos - 1; i >= 0; i-- {
		if buf[i] == '\n' {
			return i + 1
		}
	}
	return 0
}

// LineEnd returns the offset of the current line end (before \n).
func LineEnd(buf []rune, pos int) int {
	if pos < 0 {
		pos = 0
	}
	for i := pos; i < len(buf); i++ {
		if buf[i] == '\n' {
			return i
		}
	}
	return len(buf)
}
