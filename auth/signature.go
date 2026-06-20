package auth

import "strings"

type Signature struct {
	KeyID     string
	Timestamp string
	Value     string
	Algorithm string
}

func SignatureFromHeaders(headers map[string][]string) Signature {
	value := HeaderValue(headers, HeaderSignature)
	algorithm := SchemeSignature
	if prefix, rest, ok := strings.Cut(value, " "); ok {
		algorithm = strings.TrimSpace(prefix)
		value = strings.TrimSpace(rest)
	}
	return Signature{
		KeyID:     HeaderValue(headers, HeaderKeyID),
		Timestamp: HeaderValue(headers, HeaderTimestamp),
		Value:     value,
		Algorithm: algorithm,
	}
}
