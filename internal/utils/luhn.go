package utils

func ValidLuhn(v string) bool {
	if v == "" {
		return false
	}
	sum := 0
	parity := len(v) % 2
	for i := range len(v) {
		if v[i] < '0' || v[i] > '9' {
			return false
		}
		digit := int(v[i] - '0')
		if i%2 == parity {
			digit = digit * 2
			if digit > 9 {
				digit = digit - 9
			}
		}
		sum += digit
	}
	return sum%10 == 0
}
