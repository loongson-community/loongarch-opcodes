package common

import (
	"bufio"
	"os"
)

func ReadInsnDescriptionFile(path string) ([]*InsnDescription, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var result []*InsnDescription

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		l := sc.Text()

		// the line read has no newline suffix, ready for consumption

		if len(l) == 0 {
			// skip empty lines
			continue
		}

		desc, err := ParseInsnDescriptionLine(l)
		if err != nil {
			return nil, err
		}

		result = append(result, desc)
	}

	return result, nil
}
