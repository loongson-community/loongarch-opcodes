package common

func ReadInsnDescs(paths []string) ([]*InsnDescription, error) {
	var result []*InsnDescription
	for _, path := range paths {
		descs, err := ReadInsnDescriptionFile(path)
		if err != nil {
			return nil, err
		}
		result = append(result, descs...)
	}
	return result, nil
}
