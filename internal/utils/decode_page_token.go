package utils

func DecodePageToken(enocodedPageToken string) (string, error) {
	if enocodedPageToken == "" {
		return "", nil
	}

	decodedPageToken, err := DecodeBase64(enocodedPageToken)
	if err != nil {
		return "", err
	}

	return decodedPageToken, nil
}
